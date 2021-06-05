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
	ListenAddr = "api:8080"
	QueueAddr  = "amqp://guest:guest@rmq/"
)

func FileServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	imageName := vars["imageName"]
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(imageName)))
	http.ServeFile(w, r, "./output/"+id+"/"+imageName)
}

func main() {
	q := NewQueue(QueueAddr)
	defer q.conn.Close()
	defer q.ch.Close()

	s := NewServer(q)
	go q.Consume(s.Remove())
	r := mux.NewRouter()
	r.HandleFunc("/upload", s.Upload()).Methods("POST")
	r.HandleFunc("/generate", s.Generate()).Methods("POST")
	r.HandleFunc("/output/{id}/{imageName}", FileServer)
	handler := cors.Default().Handler(r)
	log.Fatal(http.ListenAndServe(ListenAddr, handler))
}
