package main

import (
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/quic-go/quic-go/http3"
)

func main() {
	var (
		port     = flag.Uint("p", 8081, "front port")
		useHttp3 = flag.Bool("enable_http3", true, "use http3")
		certPath = flag.String("c", "cert.pem", "cert file path")
		keyPath  = flag.String("k", "priv.key", "key file path")
	)
	flag.Parse()

	oproxyServer := setupHandler("/home/yequnhua/cdn/gincopy/server")
	if *useHttp3 {
		fmt.Println("http3")
		if err := http3.ListenAndServe(fmt.Sprintf("localhost:%d", *port), *certPath, *keyPath, oproxyServer); err != nil {
			log.Fatalln(err)
		}
	}
}

func setupHandler(www string) http.Handler {
	mux := http.NewServeMux()
	if len(www) > 0 {
		mux.Handle("/", http.FileServer(http.Dir(www)))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("%#v\n", r)
			const maxSize = 1 << 30 // 1 GB
			num, err := strconv.ParseInt(strings.ReplaceAll(r.RequestURI, "/", ""), 10, 64)
			if err != nil || num <= 0 || num > maxSize {
				w.WriteHeader(400)
				return
			}
			w.Write(generatePRData(int(num)))
		})
	}

	mux.HandleFunc("/demo/tile", func(w http.ResponseWriter, r *http.Request) {
		// Small 40x40 png
		w.Write([]byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
			0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x28,
			0x01, 0x03, 0x00, 0x00, 0x00, 0xb6, 0x30, 0x2a, 0x2e, 0x00, 0x00, 0x00,
			0x03, 0x50, 0x4c, 0x54, 0x45, 0x5a, 0xc3, 0x5a, 0xad, 0x38, 0xaa, 0xdb,
			0x00, 0x00, 0x00, 0x0b, 0x49, 0x44, 0x41, 0x54, 0x78, 0x01, 0x63, 0x18,
			0x61, 0x00, 0x00, 0x00, 0xf0, 0x00, 0x01, 0xe2, 0xb8, 0x75, 0x22, 0x00,
			0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
		})
	})

	mux.HandleFunc("/demo/tiles", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><head><style>img{width:40px;height:40px;}</style></head><body>")
		for i := 0; i < 200; i++ {
			fmt.Fprintf(w, `<img src="/demo/tile?cachebust=%d">`, i)
		}
		io.WriteString(w, "</body></html>")
	})

	mux.HandleFunc("/demo/echo", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error reading body while handling /echo: %s\n", err.Error())
		}
		w.Write(body)
	})

	// accept file uploads and return the MD5 of the uploaded file
	// maximum accepted file size is 1 GB
	mux.HandleFunc("/demo/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			err := r.ParseMultipartForm(1 << 30) // 1 GB
			if err == nil {
				var file multipart.File
				file, hand, err := r.FormFile("file")
				//fmt.Println(wwa.Filename)
				if err == nil {
					var size int64
					if sizeInterface, ok := file.(Size); ok {
						size = sizeInterface.Size()
						b := make([]byte, size)
						file.Read(b)
						md5 := md5.Sum(b)
						fmt.Fprintf(w, "file md5:%x\n", md5)
						file.Seek(0, 0)
						fdir := Mkdir("upload")
						fmt.Printf("upload path:%v\n", fdir+"/"+hand.Filename)
						// 创建目标文件
						destFile, err := os.Create(fdir + "/" + hand.Filename)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							fmt.Fprintf(w, "Failed to create file: %v", err)
						}
						defer destFile.Close()

						// 将上传的文件内容拷贝到目标文件
						_, err = io.Copy(destFile, file)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							fmt.Fprintf(w, "Failed to save file: %v", err)
						}

						// 返回上传成功的消息
						w.WriteHeader(http.StatusOK)
						fmt.Fprintf(w, "upload path:%v\n", fdir+"/"+hand.Filename)
						return
					}

					err = errors.New("couldn't get uploaded file size")
				}

				defer file.Close()

			}

			fmt.Printf("Error receiving upload: %#v\n", err)
			//utils.DefaultLogger.Infof("Error receiving upload: %#v", err)
		}

		io.WriteString(w, `<html><body><form action="/demo/upload" method="post" enctype="multipart/form-data">
				<input type="file" name="uploadfile"><br>
				<input type="submit">
			</form></body></html>`)
	})

	return mux
}
func Mkdir(basePath string) string {
	//	1.获取当前时间,并且格式化时间
	folderName := time.Now().Format("2006-01-02")
	folderPath := filepath.Join(basePath, folderName)
	//使用mkdirall会创建多层级目录
	os.MkdirAll(folderPath, os.ModePerm)
	return folderPath
}

// Size is needed by the /demo/upload handler to determine the size of the uploaded file
type Size interface {
	Size() int64
}

// See https://en.wikipedia.org/wiki/Lehmer_random_number_generator
func generatePRData(l int) []byte {
	res := make([]byte, l)
	seed := uint64(1)
	for i := 0; i < l; i++ {
		seed = seed * 48271 % 2147483647
		res[i] = byte(seed)
	}
	return res
}
