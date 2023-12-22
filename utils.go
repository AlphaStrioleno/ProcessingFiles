package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

// VideoExtensions 视频文件扩展名列表
var VideoExtensions = map[string]bool{
	".mp4": true, ".avi": true, ".mkv": true,
	".flv": true, ".mov": true, ".wmv": true,
	".rmvb": true, ".ts": true, ".3gp": true,
	// 添加其他视频文件扩展名
}

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

// Notice 通过pushdeer来发送完成通知到手机上
func Notice() {
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
	return VideoExtensions[ext]
}

func MakeDir(dir string) {
	// 检查文件夹是否存在
	if _, err := os.Stat(dir); err == nil {
		err := os.Mkdir(dir, fs.ModePerm)
		if err != nil {
			println("创建文件夹失败:", dir)
		}
	}

}

func RemoveFile(path string) {
	err := os.Remove(path)
	if err != nil {
		println("删除文件失败:", path)
	}
}
func RenameMove(oldPath string, newPath string) {
	err := os.Rename(oldPath, newPath)
	println("移动文件:", oldPath, "=>", newPath)
	if err != nil {
		println("重命名文件失败:", oldPath)
	}
}

func ReadJSON(path string) []byte {
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
	size := fileInfo.Size() // 文件大小，单位为字节

	b := make([]byte, size)
	_, err = file.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	return b

	//data := make(map[string]FileInfo)
	//err = json.Unmarshal(b, &data)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//return data
}

func WriteJSON(path string, data interface{}) {
	// 创建文件
	file, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	// 将数据写入文件
	b, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}
	_, err = file.Write(b)
	if err != nil {
		log.Fatal(err)
	}
}

func CheckAndDeleteEmpty(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	// 如果文件夹不为空，递归检查其子文件夹
	if len(files) > 0 {
		for _, file := range files {
			if file.IsDir() {
				CheckAndDeleteEmpty(filepath.Join(dir, file.Name()))
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
