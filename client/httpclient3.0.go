package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/quic-go/quic-go/http3"
)

func main() {

	var (
		serverAddr     = flag.String("u", "https://localhost:8081/demo/upload", "upload server address")
		uploadFilePath = flag.String("f", "httpclient3.0.go", "upload file path")
	)
	flag.Parse()
	// 创建基于QUIC的Transport
	transport := &http3.RoundTripper{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 忽略证书验证
	}

	fmt.Printf("serverAddr:%v,uploadFilePath:%v\n", *serverAddr, *uploadFilePath)
	// 创建客户端
	client := &http.Client{
		Transport: transport,
	}

	// 创建一个带有缓冲区的字节流来存储文件内容
	fileContents := bytes.NewBuffer([]byte{})

	// 打开待上传的文件
	file, err := os.Open(*uploadFilePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	// 将文件内容读取到字节流中
	_, err = fileContents.ReadFrom(file)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 创建一个带有缓冲区的字节流来存储HTTP请求体
	body := &bytes.Buffer{}

	// 创建一个multipart writer来编写multipart/form-data格式的请求体
	writer := multipart.NewWriter(body)

	// 添加待上传的文件到请求体中
	fileWriter, err := writer.CreateFormFile("file", *uploadFilePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = fileContents.WriteTo(fileWriter)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 完成写入请求体
	err = writer.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 创建一个HTTP POST请求并设置请求头和请求体
	request, err := http.NewRequest("POST", *serverAddr, body)
	if err != nil {
		fmt.Println(err)
		return
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	//req.Header.Set("Content-Type", file.FormDataContentType())
	// 发送请求并获取响应
	start := time.Now()
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body2, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// 输出响应结果
	fmt.Printf("Response: %s\n", body2)
	fmt.Printf("Upload completed in %s\n", time.Since(start))
}
