package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Data 数据结构
type Data struct {
	Filename string `json:"filename"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	HomePage string `json:"homepage"`
}

// FileInfo 文件信息
type FileInfo struct {
	Filename string `json:"filename"`
}

// VideoExtensions 视频文件扩展名列表
var VideoExtensions = map[string]bool{
	".mp4": true, ".avi": true, ".mkv": true,
	".flv": true, ".mov": true, ".wmv": true,
	".rmvb": true, ".ts": true, ".m2ts": true,
	// 添加其他视频文件扩展名
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

// IsVideoFile 检查文件是否为视频文件
func IsVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return VideoExtensions[ext]
}

// MakeDir 创建文件夹
func MakeDir(dir string) {
	// 检查文件夹是否存在
	if _, err := os.Stat(dir); err == nil {
		return
	}
	err := os.MkdirAll(dir, fs.ModePerm)
	if err != nil {
		println("创建文件夹失败:", dir, err)
	}

}

// RemoveFile 删除文件
func RemoveFile(path string) {
	err := os.Remove(path)
	println("删除文件:", path)
	if err != nil {
		println("删除文件失败:", path)
	}
}

// RenameMove 重命名/移动文件
func RenameMove(oldPath string, newPath string) {
	err := os.Rename(oldPath, newPath)
	println("重命名/移动文件:", oldPath, "=>", newPath)
	if err != nil {
		println("重命名/移动文件失败:", oldPath, err)
	}
}

// ReadJSON 读取json文件
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

}

// CheckAndDeleteEmpty 检查并删除空文件夹
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

// IsLessThan120MB 检查文件大小是否小于120MB
func IsLessThan120MB(path string) bool {
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

// PathSet 获取文件夹下的文件/文件夹列表
func PathSet(sourcePath string, kind string) map[string]bool {
	path, err := os.ReadDir(sourcePath)
	if err != nil {
		log.Fatal(err)
	}
	pathSet := make(map[string]bool)
	for _, f := range path {
		if kind == "file" {
			if !f.IsDir() {
				pathSet[f.Name()] = true
			}
		} else if kind == "folder" {
			if f.IsDir() {
				pathSet[f.Name()] = true
			}
		}
	}
	return pathSet
}
