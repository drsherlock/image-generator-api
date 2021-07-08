package main

import (
	"encoding/json"
	"fmt"
	// "github.com/drsherlock/image-generator-api/compress"
	"github.com/drsherlock/imagegen"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	FileSizeErr   = "The uploaded file is too big. Please choose an file that's less than 5MB in size"
	FileFormatErr = "The provided file format is not allowed. Please upload a JPEG/JPG/PNG image"
)
const MAX_UPLOAD_SIZE = 1024 * 1024 * 5 // 5 MB

type server struct{ queue *queue }

func NewServer(queue *queue) *server {
	return &server{queue: queue}
}

type ImageId struct {
	Id string `json:"id"`
}

type ImageName struct {
	Name string
}

func (s *server) Check() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}
}

func (s *server) Upload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MAX_UPLOAD_SIZE)
		if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
			http.Error(w, FileSizeErr, http.StatusBadRequest)
			return
		}

		file, fileHeader, err := r.FormFile("image")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		buff := make([]byte, 512)
		if _, err := file.Read(buff); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fileType := http.DetectContentType(buff)
		if !isValidFileExt(filepath.Ext(fileHeader.Filename)) || !isValidFileType(fileType) {
			http.Error(w, FileFormatErr, http.StatusBadRequest)
			return
		}

		if _, err := file.Seek(0, io.SeekStart); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		id := fmt.Sprint(time.Now().UnixNano())
		dst, err := os.Create(fmt.Sprintf("./uploads/%s%s", id, filepath.Ext(fileHeader.Filename)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		imageId := ImageId{
			Id: id,
		}

		imageName := ImageName{
			Name: id + filepath.Ext(fileHeader.Filename),
		}

		if err := s.queue.Publish(imageName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(imageId); err != nil {
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
		if err := json.NewDecoder(r.Body).Decode(&image); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		file := new(os.File)
		if err := filepath.Walk("uploads", findFile(image.FileId, file)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := imagegen.Create(file, image.Title, image.TitleColor, image.Fonts); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var images []string
		fileName := strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name()))
		if err := filepath.Walk("output/"+fileName, visitDir(&images)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// fileName := strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name()))
		// if err = compress.ZipFiles("output/"+fileName+".zip", "output/"+fileName); err != nil {
		// 	http.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		imPath := struct {
			Images []string `json:"images"`
		}{
			Images: images,
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")
		// w.Header().Set("Content-Disposition", "attachment; filename="+fileName+".zip")
		// w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(imPath); err != nil {
			panic(err)
		}
		// compressedFile, err := os.Open("output/" + fileName + ".zip")
		// if err != nil {
		// 	http.Error(w, err.Error(), http.StatusBadRequest)
		// 	return
		// }
		// defer compressedFile.Close()
		// io.Copy(w, compressedFile)
	}
}

func (s *server) Remove() func(imageName *ImageName) error {
	return func(imageName *ImageName) error {
		if err := os.RemoveAll("./output/" + strings.TrimSuffix(imageName.Name, filepath.Ext(imageName.Name))); err != nil {
			return err
		}

		if err := os.Remove("./uploads/" + imageName.Name); err != nil {
			return err
		}

		return nil
	}
}

func findFile(fileId string, file *os.File) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fileName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		if !info.IsDir() && isFileMatching(fileName, fileId) {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			*file = *f
		}
		return nil
	}
}

func visitDir(files *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			*files = append(*files, path)
		}
		return nil
	}
}

func isValidFileType(fileType string) bool {
	if fileType != "image/jpeg" && fileType != "image/jpg" && fileType != "image/png" {
		return false
	}

	return true
}

func isValidFileExt(fileExt string) bool {
	if fileExt != ".jpeg" && fileExt != ".jpg" && fileExt != ".png" {
		return false
	}

	return true
}

func isFileMatching(fileName string, fileId string) bool {
	return fileName == fileId
}
