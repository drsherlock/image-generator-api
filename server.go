package main

import (
	"encoding/json"
	"github.com/drsherlock/image-generator-api/compress"
	// "errors"
	"fmt"
	"github.com/drsherlock/imagegen"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	// "context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ListenAddr = "localhost:8080"
)

// type generateRequest struct {
// 	FileId int64    `json:"fileId"`
// 	Color  string   `json:"color"`
// 	Fonts  []string `json:"fonts"`
// }

type server struct{}

// var ctx = context.Background()

func (s *server) Upload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, fileHeader, err := r.FormFile("image")
		if err != nil {
			fmt.Println("ERROR", err)
		}
		defer file.Close()

		err = os.MkdirAll("./uploads", os.ModePerm)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fileId := time.Now().UnixNano()
		dst, err := os.Create(fmt.Sprintf("./uploads/%d%s", fileId, filepath.Ext(fileHeader.Filename)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		image := struct {
			FileId int64 `json:"fileId"`
		}{
			FileId: fileId,
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(image); err != nil {
			panic(err)
		}
	}
}

func (s *server) Generate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		image := struct {
			FileId     string   `json:"fileId"`
			Title      string   `json:"title"`
			TitleColor string   `json:"titleColor"`
			Fonts      []string `json:"fonts"`
		}{}
		err := json.NewDecoder(r.Body).Decode(&image)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var file *os.File
		err = filepath.Walk("uploads", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && isFileMatching(info.Name(), image.FileId) {
				file, err = os.Open(path)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		imagegen.Create(file, image.Title, image.TitleColor, image.Fonts)

		fileName := strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name()))
		err = compress.ZipFiles("output/"+fileName+".zip", "output/"+fileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "hello world"}`))
	}
}

func isFileMatching(fileName string, fileId string) bool {
	return fileName == fileId+".jpg" || fileName == fileId+".jpeg" || fileName == fileId+".png"
}

// func findImage(fileId string, file *os.File) func(path string, info os.FileInfo, err error) error {
// 	return func(path string, info os.FileInfo, err error) error {
// 		if err != nil {
// 			return err
// 		}
//
// 		if !info.IsDir() && isFileMatching(info.Name(), fileId) {
// 			file, err = os.Open(path)
// 			if err != nil {
// 				return err
// 			}
// 		}
// 		return nil
// 	}
// }

func main() {
	s := &server{}
	r := mux.NewRouter()
	r.HandleFunc("/upload", s.Upload()).Methods("POST")
	r.HandleFunc("/generate", s.Generate()).Methods("POST")
	handler := cors.Default().Handler(r)
	log.Fatal(http.ListenAndServe(ListenAddr, handler))
}
