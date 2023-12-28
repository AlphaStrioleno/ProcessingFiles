package main

import (
	"encoding/json"
	"fmt"
	"github.com/metatube-community/metatube-sdk-go/engine"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CleanFile 清理文件夹
func CleanFile(sourcePath string) {
	var filesToMove []string

	println("开始清理文件夹:", sourcePath)
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Dir(path) != sourcePath {
			if !IsVideoFile(path) || IsLessThan120MB(path) {
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

// CreateNamesJSON 生成output.json
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
			data[filename] = map[string]string{"filename": name}
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Create("output.json")
	if err != nil {
		log.Fatal(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)
	_, err = file.Write(jsonData)
	if err != nil {
		log.Fatal(err)
	}
}

// RenameFile 重命名文件
func RenameFile(sourcePath string) {
	// 读取JSON文件
	data := map[string]FileInfo{}
	// 读取json文件
	r := ReadJSON("output.json")
	err := json.Unmarshal(r, &data)
	if err != nil {

	}
	fileSet := PathSet(sourcePath, "file")
	for nameWithSuffix, fileInfo := range data {
		if fileSet[nameWithSuffix] {
			videoPath := filepath.Join(sourcePath, nameWithSuffix)
			if fileInfo.Filename == "d" {
				RemoveFile(videoPath)
			} else if fileInfo.Filename == "m" {
				laterPath := filepath.Join(sourcePath, "Later")
				MakeDir(laterPath)
				newPath := filepath.Join(laterPath, nameWithSuffix)
				RenameMove(videoPath, newPath)
			} else {
				newPath := filepath.Join(sourcePath, fileInfo.Filename+filepath.Ext(nameWithSuffix))
				RenameMove(videoPath, newPath)
			}
		}
	}
}

// GetNumber 获取号码
func GetNumber(sourcePath string) {
	app := engine.Default()

	file, err := os.ReadDir(sourcePath)
	if err != nil {
		log.Fatal(err)
	}
	dataMap := make(map[string]Data)
	for _, f := range file {
		if f.IsDir() {
			continue
		}
		// 创建一个空的Data结构体
		data := Data{}
		fileName := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
		data.Filename = fileName
		results, err := app.SearchMovieAll(fileName, true)
		if err != nil {
			log.Printf("搜索失败: %v", err)
			continue
		}
		if len(results) == 0 {
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
		number := result.Number
		filePath := filepath.Join(sourcePath, f.Name())
		matches := regexp.MustCompile(`cd(\d+)$`).FindStringSubmatch(strings.ToLower(fileName))
		if len(matches) > 1 {
			number = number + matches[1]
		}
		newPath := filepath.Join(sourcePath, number+filepath.Ext(f.Name()))
		RenameMove(filePath, newPath)
		data.Path = newPath
		data.HomePage = result.Homepage

		time.Sleep(3 * time.Second)
		// 创建一个map，key是id，value是Data结构体

		// 将Data结构体添加到map中，key是id
		dataMap[number] = data
	}
	// 将map转换为JSON
	jsonData, err := json.Marshal(dataMap)
	if err != nil {
		log.Fatal(err)
	}
	// 将JSON写入到文件中
	j, err := os.Create("data.json")
	if err != nil {
		log.Fatal(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(j)
	_, err = j.Write(jsonData)
	if err != nil {
		log.Fatal(err)
	}
	Notice()
}

// MoveFile 移动文件
func MoveFile(sourcePath string, destPath string) {
	// 读取json文件
	data := make(map[string]Data)
	//r := ReadJSON("output.json", data).(map[string]Data)
	r := ReadJSON("data.json")
	err := json.Unmarshal(r, &data)
	if err != nil {
		log.Fatal(err)
	}
	// 遍历data
	for number, information := range data {
		name := information.Name
		// 如果Name为空，跳过
		if name == "" {
			continue
		}
		nameFolder := filepath.Join(destPath, name)
		// 如果name文件夹不存在，创建name文件夹，存在跳过
		MakeDir(nameFolder)
		destNumberFolder := filepath.Join(sourcePath, number)
		destFilePath := filepath.Join(destNumberFolder, number+filepath.Ext(filepath.Base(information.Path)))

		destNumberFolderSet := PathSet(nameFolder, "folder")
		if destNumberFolderSet[number] {
			// 如果number文件夹已经存在，检查第number文件夹里的文件和data.json里的文件是否一致
			// 如果有相同的视频文件，删除原来的文件
			if _, err := os.Stat(destFilePath); err == nil {
				RemoveFile(destFilePath)
			}
		} else {
			// 如果number文件夹不存在，创建number文件夹
			MakeDir(destNumberFolder)
		}
		RenameMove(information.Path, destFilePath)

	}
}

// Helper 帮助
func Helper() {
	fmt.Println("c: 清理并移动文件")
	fmt.Println("j: 获取名称并生成output.json以便手动重命名, 使用tsz命令下载本地")
	fmt.Println("r: 使用trz命令上传, 根据output.json文件来重命名")
	fmt.Println("n: 使用metatube来获取信息生成data.json")
	fmt.Println("f: 根据data.json创建文件夹并移动文件")
	fmt.Println("e: 清理目标文件夹中的空文件夹")
	return
}

// Run 运行
func Run() {
	// 获取命令行参数
	args := os.Args

	sourcePath := ""
	destPath := ""
	if len(args) == 1 {
		Helper()
		return
	} else if len(args) == 2 {
		// 获取当前用户
		currentUser, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		// 获取用户文件夹路径
		homeDir := currentUser.HomeDir
		sourcePath = filepath.Join(homeDir, "media/Further")
		destPath = filepath.Join(homeDir, "media/output")
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
		MoveFile(sourcePath, destPath)
	} else if args[1] == "c" {
		CleanFile(sourcePath)
	} else if args[1] == "j" {
		CreateNamesJSON(sourcePath)
	} else if args[1] == "r" {
		RenameFile(sourcePath)
	} else if args[1] == "e" {
		CheckAndDeleteEmpty(destPath)
	} else {
		Helper()
	}
}

func main() {
	Run()
}

//
