package main

import (
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"log"
	"mime"
	"net/http"
	"path/filepath"
)

const (
	ListenAddr = "localhost:8080"
)

func FileServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	imageId := vars["imageId"]
	imageName := vars["imageName"]
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(imageName)))
	http.ServeFile(w, r, "./output/"+imageId+"/"+imageName)
}

func main() {
	s := NewServer()
	r := mux.NewRouter()
	r.HandleFunc("/upload", s.Upload()).Methods("POST")
	r.HandleFunc("/generate", s.Generate()).Methods("POST")
	r.HandleFunc("/output/{imageId}/{imageName}", FileServer)
	handler := cors.Default().Handler(r)
	log.Fatal(http.ListenAndServe(ListenAddr, handler))
}
