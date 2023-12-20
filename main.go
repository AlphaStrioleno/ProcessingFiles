package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/metatube-community/metatube-sdk-go/engine"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Data struct {
	Filename string `json:"filename"`
	Name     string `json:"name"`
	HomePage string `json:"homepage"`
}

type FileInfo struct {
	Filename string `json:"filename"`
}

// Folder represents a folder in the filesystem with its name and sequence number
type Folder struct {
	Name      string
	SeqNumber int
}

// Folders is a slice of Folder
type Folders []Folder

func (f Folders) Len() int           { return len(f) }
func (f Folders) Less(i, j int) bool { return f[i].SeqNumber > f[j].SeqNumber }
func (f Folders) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }

// GetFolders now returns a map with the base folder name as the key and a list of its variations as the value
func GetFolders(dir string) (map[string]Folders, error) {
	folderMap := make(map[string]Folders)

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			var baseName string
			var seqNumber int

			if strings.Contains(file.Name(), "(") {
				baseName = strings.Split(file.Name(), "(")[0]
				seqNumber, _ = strconv.Atoi(strings.TrimLeft(strings.Split(file.Name(), "(")[1], ")"))
			} else {
				baseName = file.Name()
				seqNumber = -1
			}

			folder := Folder{
				Name:      file.Name(),
				SeqNumber: seqNumber,
			}

			folderMap[baseName] = append(folderMap[baseName], folder)
		}
	}

	for _, folders := range folderMap {
		sort.Sort(folders)
	}

	return folderMap, nil
}

// 视频文件扩展名列表
var videoExtensions = map[string]bool{
	".mp4": true, ".avi": true, ".mkv": true,
	".flv": true, ".mov": true, ".wmv": true,
	".rmvb": true, ".ts": true, ".3gp": true,
	// 添加其他视频文件扩展名
}

// 通过bark来发送完成通知到手机上
func notice() {
	url := "https://api2.pushdeer.com/message/push?pushkey=PDU21180THMbOUv9clNBH810DysiU4SsDRsC4cRUs&text=已经完成了"
	payload := []byte("")

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println("请求创建失败:", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("请求发送失败:", err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(resp.Body)

	fmt.Println("响应状态码:", resp.StatusCode)
}

// 检查文件是否为视频文件
func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return videoExtensions[ext]
}

// 检查文件大小是否小于120MB
func isLessThan120MB(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	fileInfo, _ := file.Stat()
	size := fileInfo.Size()      // 文件大小，单位为字节
	sizeMB := size / 1024 / 1024 // 文件大小，单位为MB

	return sizeMB < 120
}

func MoveFolders(srcDir string, tgtDir string, parentDir string) error {
	files, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			dstDir := tgtDir
			if _, err := os.Stat(tgtDir + "/" + file.Name()); err == nil {
				dstDir = parentDir
			}

			// Check if the destination directory exists, if not create it
			if _, err := os.Stat(dstDir); os.IsNotExist(err) {
				err := os.Mkdir(dstDir, 0755)
				if err != nil {
					return err
				}
			}

			err := os.Rename(srcDir+"/"+file.Name(), dstDir+"/"+file.Name())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// 处理重复文件夹
func handleDuplicateFolder(sourcePath string) {
	folderMap, err := GetFolders(sourcePath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, folders := range folderMap {
		baseFolder := folders[len(folders)-1]
		for i := 0; i < len(folders)-1; i++ {
			err := MoveFolders(sourcePath+"/"+folders[i].Name, sourcePath+"/"+baseFolder.Name, sourcePath)
			println("移动文件夹:", sourcePath+"/"+folders[i].Name, "=>", sourcePath+"/"+baseFolder.Name)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
		}
	}
	checkAndDeleteEmpty(sourcePath)
}

func checkAndDeleteEmpty(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	// 如果文件夹不为空，递归检查其子文件夹
	if len(files) > 0 {
		for _, file := range files {
			if file.IsDir() {
				checkAndDeleteEmpty(filepath.Join(dir, file.Name()))
			}
		}
	}

	// 再次检查该文件夹，如果现在为空（可能其子文件夹都被删除了），那么删除它
	files, err = os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	if len(files) == 0 {
		err := os.Remove(dir)
		println("删除文件夹:", dir)
		if err != nil {
			return
		}
	}
}

func cleanFile(sourcePath string) {
	var filesToMove []string

	println("开始清理文件夹:", sourcePath)
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Dir(path) != sourcePath {
			if !isVideoFile(path) || isLessThan120MB(path) {
				err := os.Remove(path)
				if err != nil {
					return err
				} // 删除文件
				println("删除文件:", path)
			} else {
				filesToMove = append(filesToMove, path)
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	// 移动文件
	println("开始移动文件")
	for _, oldPath := range filesToMove {
		fileName := filepath.Base(oldPath)
		newPath := filepath.Join(sourcePath, fileName)

		// 如果目标文件夹已存在同名文件，为两个文件都添加创建时间前缀
		if _, err := os.Stat(newPath); err == nil {
			// 获取目标文件夹中同名文件的创建时间，并将其格式化为字符串
			existFileInfo, err := os.Stat(newPath)
			if err != nil {
				log.Fatal(err)
			}
			// 将已经存在的同名文件重命名，添加创建时间前缀
			existFileCreateTime := existFileInfo.ModTime().Format("20060102150405")
			existFileNewName := existFileCreateTime + "_" + fileName
			err = os.Rename(newPath, filepath.Join(sourcePath, existFileNewName))
			println("重命名已经存在的文件:", newPath, "=>", filepath.Join(sourcePath, existFileNewName))
			if err != nil {
				log.Fatal(err)
			}

			// 对要移动的文件做同样的处理
			toMoveFileInfo, err := os.Stat(oldPath)
			if err != nil {
				log.Fatal(err)
			}
			toMoveFileCreateTime := toMoveFileInfo.ModTime().Format("20060102150405")
			toMoveFileNewName := toMoveFileCreateTime + "_" + fileName
			newPath = filepath.Join(sourcePath, toMoveFileNewName)
		}

		err := os.Rename(oldPath, newPath)
		if err != nil {
			log.Fatal(err)
		} // 移动文件
		println("移动文件:", oldPath, "=>", newPath)
	}

	// 删除空文件夹
	println("开始删除空文件夹")
	checkAndDeleteEmpty(sourcePath)
}

func createNamesJSON(sourcePath string) {
	files, err := os.ReadDir(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	data := make(map[string]map[string]string)
	for _, file := range files {
		if !file.IsDir() {
			filename := file.Name()
			name := strings.TrimSuffix(filename, filepath.Ext(filename))
			data[name] = map[string]string{"filename": name}
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取当前目录地址失败：", err)
		return
	}
	out := filepath.Join(dir, "output.json")
	outputFile, err := os.Create(out)
	if err != nil {
		log.Fatal(err)
	}
	defer func(outputFile *os.File) {
		err := outputFile.Close()
		if err != nil {

		}
	}(outputFile)

	_, err = outputFile.Write(jsonData)
	if err != nil {
		log.Fatal(err)
	}
}

func renameFile(sourcePath string) {
	// 读取JSON文件
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取当前目录地址失败：", err)
		return
	}
	out := filepath.Join(dir, "output.json")
	file, err := os.ReadFile(out)
	if err != nil {
		panic(err)
	}

	// 解析JSON文件
	filesToRename := map[string]FileInfo{}
	err = json.Unmarshal(file, &filesToRename)
	if err != nil {
		panic(err)
	}

	// 遍历指定的文件夹
	err = filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 如果是文件
		if !d.IsDir() {
			fileNameWithoutExt := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))

			// 如果在JSON中找到键
			if fileInfo, exists := filesToRename[fileNameWithoutExt]; exists {
				if fileInfo.Filename == "d" {
					// 删除文件
					err = os.Remove(path)
					fmt.Println("删除文件:", path)
					if err != nil {
						return err
					}
				} else if fileInfo.Filename == "m" {
					// 移动文件到指定文件夹
					laterPath := filepath.Join(filepath.Dir(path), "Later")
					if _, err := os.Stat(laterPath); err != nil {
						err = os.Mkdir(laterPath, fs.ModePerm)
					}
					newPath := filepath.Join(laterPath, d.Name())
					err = os.Rename(path, newPath)
					fmt.Println("移动文件:", path, "=>", newPath)
					if err != nil {
						return err
					}
				} else if fileInfo.Filename != fileNameWithoutExt {
					// 重命名文件
					newPath := filepath.Join(filepath.Dir(path), fileInfo.Filename+filepath.Ext(d.Name()))
					err = os.Rename(path, newPath)
					fmt.Println("重命名文件:", path, "=>", newPath)
					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		panic(err)
	}
}

func getNames(sourcePath string) (nameList []string, err error) {
	files, err := os.ReadDir(sourcePath) // 读取文件夹
	if err != nil {
		fmt.Println("读取文件夹失败:", err)
		return
	}

	var fileList []string // 创建一个空的字符串列表来存储文件名

	for _, file := range files {
		if !file.IsDir() { // 确保这不是一个目录
			fileName := file.Name()                   // 获取文件名
			ext := filepath.Ext(fileName)             // 获取文件后缀
			name := strings.TrimSuffix(fileName, ext) // 移除文件名的后缀
			fileList = append(fileList, name)         // 将文件名加入到列表中
		}
	}

	return fileList, nil
}

func getNumber(sourcePath string) {
	app := engine.Default()

	nameList, err := getNames(sourcePath)
	if err != nil {
		log.Fatal(err)
	}
	// 创建一个map，key是id，value是Data结构体
	dataMap := make(map[string]Data)
	for _, fileName := range nameList {
		results, err := app.SearchMovieAll(fileName, true)
		if err != nil {
			log.Printf("搜索失败: %v", err)
			continue
		}
		result := results[0]
		// 如果没有搜索结果，跳过
		for _, r := range results {
			if len(r.Actors) == 0 {
				continue
			} else {
				result = r
				break
			}
		}

		// 创建一个空的Data结构体
		data := Data{}

		// 处理name
		if len(result.Actors) == 0 {
			data.Name = "佚名"
		} else if len(result.Actors) == 1 {
			data.Name = result.Actors[0]
		} else {
			// 循环处理names, id, file
			for _, name := range result.Actors {
				// 拼接names
				data.Name += name + ","

				if len(data.Name) > 50 {
					if len(result.Actors) >= 3 {
						// 如果超过50字符且names长度大于等于3，取前三个name拼接
						data.Name = result.Actors[0] + "," + result.Actors[1] + "," + result.Actors[2]
						if len(data.Name) > 50 {
							// 如果还是超过50字符，取前50个字符
							data.Name = data.Name[:50]
						}
					} else {
						// 如果names长度小于3，直接取前50个字符
						data.Name = data.Name[:50]
					}

				}
			}
		}
		data.Filename = fileName
		data.HomePage = result.Homepage

		time.Sleep(3 * time.Second)
		// 将Data结构体添加到map中，key是id
		dataMap[result.Number] = data
	}
	// 将map转换为JSON
	jsonData, err := json.Marshal(dataMap)
	if err != nil {
		log.Fatal(err)
	}
	// 将JSON写入到文件中
	f, _ := os.OpenFile("data.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {

		}
	}(f)
	_, err = f.Write(jsonData)
	if err != nil {
		log.Fatal(err)
	}
	notice()
}

func moveFile(sourcePath string) {
	// 读取json文件
	jsonData, err := os.ReadFile("data.json")
	if err != nil {
		log.Printf("读取json文件失败: %v", err)
		return
	}
	// 解析json数据
	data := make(map[string]Data)
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		log.Printf("解析json数据失败: %v", err)
		return
	}
	// 遍历json数据，处理文件
	for k, v := range data {
		// 如果Name为空，跳过
		if v.Name == "" {
			continue
		}
		// 创建文件夹
		newDirPath := filepath.Join(sourcePath, v.Name, k)
		err := os.MkdirAll(newDirPath, fs.ModePerm)
		if err != nil {
			log.Printf("创建文件夹失败: %v", err)
			continue
		}

		// 查找并移动文件
		err = filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() || filepath.Dir(path) != sourcePath {
				return nil
			}

			name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))

			if name == v.Filename {
				newPath := filepath.Join(newDirPath, k+filepath.Ext(d.Name()))
				err = os.Rename(path, newPath)
				fmt.Println("移动文件:", path, "=>", newPath)
				time.Sleep(2 * time.Second)
				if err != nil {
					log.Printf("移动文件失败: %v", err)
				}
			}
			return nil
		})

		if err != nil {
			log.Printf("查找文件失败: %v", err)
			continue
		}
	}
}

func helper() {
	fmt.Println("c: 清理并移动文件")
	fmt.Println("j: 获取名称并生成output.json以便手动重命名, 使用tsz命令下载本地")
	fmt.Println("r: 使用trz命令上传, 根据output.json文件来重命名")
	fmt.Println("n: 使用metatube来获取信息生成data.json")
	fmt.Println("f: 根据data.json创建文件夹并移动文件")
	return
}

func renameByNumber(sourcePath string) {
	err := filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !isVideoFile(path) {
			return nil
		}

		fileName := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		folderName := filepath.Base(filepath.Dir(path))
		if fileName != folderName {
			newPath := filepath.Join(filepath.Dir(path), folderName+filepath.Ext(d.Name()))
			err = os.Rename(path, newPath)
			fmt.Println("重命名文件:", path, "=>", newPath)
			if err != nil {
				log.Printf("重命名文件失败: %v", err)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("查找文件失败: %v", err)
	}
}

func main() {
	// 获取命令行参数
	args := os.Args

	sourcePath := ""
	if len(args) == 1 {
		helper()
		return
	} else if len(args) == 2 {
		sourcePath = "/root/media"
	} else {
		sourcePath = args[2]
		_, err := os.Stat(sourcePath)

		if err == nil {
			fmt.Println("开始处理文件夹:", sourcePath)
		} else if os.IsNotExist(err) {
			fmt.Println("文件夹不存在")
			return
		} else {
			fmt.Println("发生错误:", err)
			return
		}
	}

	if args[1] == "n" {
		getNumber(sourcePath)
	} else if args[1] == "f" {
		moveFile(sourcePath)
	} else if args[1] == "c" {
		cleanFile(sourcePath)
	} else if args[1] == "j" {
		createNamesJSON(sourcePath)
	} else if args[1] == "r" {
		renameFile(sourcePath)
	} else if args[1] == "d" {
		handleDuplicateFolder(sourcePath)
	} else if args[1] == "n" {
		renameByNumber(sourcePath)
	} else {
		helper()
	}
}

//
