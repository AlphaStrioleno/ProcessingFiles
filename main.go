package main

import (
	"encoding/json"
	"fmt"
	"github.com/metatube-community/metatube-sdk-go/engine"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func MoveFolders(srcDir string, tgtDir string, parentDir string) {
	files, err := os.ReadDir(srcDir)
	if err != nil {
		println("Error:", err)
	}

	for _, file := range files {
		if file.IsDir() {
			dstDir := tgtDir
			if _, err := os.Stat(tgtDir + "/" + file.Name()); err == nil {
				dstDir = parentDir
			}

			// Check if the destination directory exists, if not create it
			MakeDir(dstDir)
			RenameMove(srcDir+"/"+file.Name(), dstDir+"/"+file.Name())
		}
	}

}

// HandleDuplicateFolder 处理重复文件夹
func HandleDuplicateFolder(sourcePath string) {
	folderMap, err := GetFolders(sourcePath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, folders := range folderMap {
		baseFolder := folders[len(folders)-1]
		for i := 0; i < len(folders)-1; i++ {
			MoveFolders(sourcePath+"/"+folders[i].Name, sourcePath+"/"+baseFolder.Name, sourcePath)
		}
	}
	CheckAndDeleteEmpty(sourcePath)
}

func CleanFile(sourcePath string) {
	var filesToMove []string

	println("开始清理文件夹:", sourcePath)
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Dir(path) != sourcePath {
			if !isVideoFile(path) || isLessThan120MB(path) {
				RemoveFile(path)
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

		RenameMove(oldPath, newPath)
	}

	// 删除空文件夹
	println("开始删除空文件夹")
	CheckAndDeleteEmpty(sourcePath)
}

func CreateNamesJSON(sourcePath string) {
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

	WriteJSON("output.json", jsonData)
}

func RenameFile(sourcePath string) {
	// 读取JSON文件
	data := map[string]FileInfo{}
	j := ReadJSON("output.json", data).(map[string]interface{})
	// 遍历指定的文件夹
	err := filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 如果是文件
		if !d.IsDir() {
			file := d.Name()
			ext := filepath.Ext(file)
			nameWithoutSuffix := strings.TrimSuffix(file, ext)
			// 如果在JSON中找到键
			if fileInfo, exists := j[nameWithoutSuffix]; exists {
				fileName := fileInfo.(map[string]interface{})["Filename"].(string)

				if fileName == "d" {
					RemoveFile(path)
				} else if fileName == "m" {
					laterPath := filepath.Join(filepath.Dir(path), "Later")
					MakeDir(laterPath)
					newPath := filepath.Join(laterPath, d.Name())
					RenameMove(path, newPath)
				} else if fileName != nameWithoutSuffix {
					newPath := filepath.Join(filepath.Dir(path), fileName+ext)
					RenameMove(path, newPath)
				}

			}
		}

		return nil

	})
	if err != nil {
		panic(err)
	}
}

func GetFileNames(sourcePath string) (nameList []string, err error) {
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

func GetNumber(sourcePath string) {
	app := engine.Default()

	nameList, err := GetFileNames(sourcePath)
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
	WriteJSON("data.json", jsonData)
	Notice()
}

func MoveFile(sourcePath string) {
	// 读取json文件
	data := make(map[string]Data)
	r := ReadJSON("output.json", data).(map[string]interface{})

	for k, v := range r {
		// 如果Name为空，跳过
		value := v.(map[string]interface{})
		if value["Name"] == "" {
			continue
		}
		// 创建文件夹
		newDirPath := filepath.Join(sourcePath, value["Name"].(string), k)
		MakeDir(newDirPath)

		// 查找并移动文件
		err := filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() || filepath.Dir(path) != sourcePath {
				return nil
			}

			name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))

			if name == value["Filename"].(string) {
				matches := regexp.MustCompile(`cd(\d+)$`).FindStringSubmatch(strings.ToLower(name))
				if len(matches) > 1 {
					k = k + matches[1]
				}
				newPath := filepath.Join(newDirPath, k+filepath.Ext(d.Name()))
				RenameMove(path, newPath)
				time.Sleep(2 * time.Second)
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

func GetFolderJSON(sourcePath string) {
	filesMap := make(map[string]string)

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理第二层的文件或文件夹
		if len(filepath.SplitList(path)) == 2 {
			filesMap[info.Name()] = path
		}

		return nil
	})

	if err != nil {
		fmt.Println("遍历文件夹出错：", err)
	}

	result, err := json.Marshal(filesMap)
	if err != nil {
		fmt.Println("JSON 序列化出错：", err)
	}

	WriteJSON("folders.json", result)
}

func RenameByNumber(sourcePath string) {
	err := filepath.WalkDir(sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !isVideoFile(path) {
			return nil
		}

		fileName := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		newStr := regexp.MustCompile(`cd\d+$`).ReplaceAllString(fileName, "")
		if newStr != fileName {
			fileName = newStr
		}
		folderName := filepath.Base(filepath.Dir(path))
		if fileName != folderName {
			newPath := filepath.Join(filepath.Dir(path), folderName+filepath.Ext(d.Name()))
			RenameMove(path, newPath)
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
		GetNumber(sourcePath)
	} else if args[1] == "f" {
		MoveFile(sourcePath)
	} else if args[1] == "c" {
		CleanFile(sourcePath)
	} else if args[1] == "j" {
		CreateNamesJSON(sourcePath)
	} else if args[1] == "r" {
		RenameFile(sourcePath)
	} else if args[1] == "d" {
		HandleDuplicateFolder(sourcePath)
	} else if args[1] == "n" {
		RenameByNumber(sourcePath)
	} else if args[1] == "o" {
		GetFolderJSON(sourcePath)
	} else {
		helper()
	}
}

//
