package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/metatube-community/metatube-sdk-go/engine"
)

type Data struct {
	Filename string `json:"filename"`
	Name     string `json:"name"`
}

type FileInfo struct {
	Filename string `json:"filename"`
}

// 视频文件扩展名列表
var videoExtensions = map[string]bool{
	".mp4": true, ".avi": true, ".mkv": true,
	".flv": true, ".mov": true, ".wmv": true,
	// 添加其他视频文件扩展名
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

func cleanFile(sourcePath string) {
	var filesToMove []string

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
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
	for _, oldPath := range filesToMove {
		newName := filepath.Base(oldPath)
		newPath := filepath.Join(sourcePath, newName)

		// 如果目标文件夹已存在同名文件，为两个文件都添加创建时间前缀
		if _, err := os.Stat(newPath); err == nil {
			// 获取目标文件夹中同名文件的创建时间，并将其格式化为字符串
			fileInfo, err := os.Stat(newPath)
			if err != nil {
				log.Fatal(err)
			}
			createTime := fileInfo.ModTime().Format("20060102150405")
			newName = createTime + "_" + newName
			newPath = filepath.Join(sourcePath, newName)

			// 对要移动的文件做同样的处理
			fileInfo, err = os.Stat(oldPath)
			if err != nil {
				log.Fatal(err)
			}
			createTime = fileInfo.ModTime().Format("20060102150405")
			err = os.Rename(oldPath, filepath.Join(filepath.Dir(oldPath), createTime+"_"+filepath.Base(oldPath)))
			if err != nil {
				return
			}
		}

		err := os.Rename(oldPath, newPath)
		if err != nil {
			return
		} // 移动文件
		println("移动文件:", oldPath, "=>", newPath)
	}

	// 删除空文件夹
	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 如果是空文件夹
		if info.IsDir() {
			// 读取文件夹
			files, err := os.ReadDir(path)
			if err != nil {
				return err
			}

			// 如果文件夹为空
			if len(files) == 0 {
				err := os.Remove(path)
				if err != nil {
					return err
				} // 删除文件夹
				println("删除文件夹:", path)
			}
		}

		return nil

	})
	if err != nil {
		log.Fatal(err)
	}
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

	outputFile, err := os.Create("output.json")
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
	file, err := os.ReadFile("output.json")
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
		// 创建一个空的Data结构体
		data := Data{}

		// 	data.File = fileName
		if len(result.Actors) == 0 {
			data.Name = "佚名"
		} else if len(result.Actors) == 1 {
			data.Name = result.Actors[0]
		} else {
			// 循环处理names, id, file
			for _, name := range result.Actors {
				// 拼接names
				data.Name += name + "、"

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

			if d.IsDir() {
				return nil
			}

			if strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) == v.Filename {
				newPath := filepath.Join(newDirPath, filepath.Base(path))
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

func main() {
	// 获取命令行参数
	args := os.Args

	if args[1] == "h" {
		fmt.Println("Usage: main [n|f] [path]")
		fmt.Println("c: 清理并移动文件")
		fmt.Println("j: 获取名称并生成json文件以便手动重命名")
		fmt.Println("r: 根据json文件来重命名")
		fmt.Println("n: 使用metatube来获取名字")
		fmt.Println("f: 创建文件夹并移动文件")
		return
	}
	sourcePath := ""
	if len(args) < 3 {
		sourcePath = "/root/media"
	} else {
		sourcePath = args[2]
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
	}
}
