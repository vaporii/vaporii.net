package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"time"
	// "html/template"
)

type Status struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func liveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	// w.Header().Set("X-Content-Type-Options", "nosniff")
	// w.Header().Set("Transfer-Encoding", "chunked")
	// w.Header().Set("Connection", "keep-alive")

	for {
		fmt.Fprintf(w, "<p>%s</p>\r\n", randomString(10))
		w.(http.Flusher).Flush()
		time.Sleep(time.Second)
	}
}

func statusPage(w http.ResponseWriter, r *http.Request) {
	data := Status{
		Message:   randomString(10),
		Timestamp: time.Now().Format("jan 2, 3:04 pm"),
	}

	tmpl, err := template.ParseFiles("./templates/status.html")
	if err != nil {
		log.Fatal("error loading template: ", err)
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Fatal("error rendering template: ", err)
	}
}

func main() {
	fs := http.FileServer(http.Dir("./public"))

	http.Handle("/", http.StripPrefix("/", fs))
	http.HandleFunc("/status", liveHandler)

	port := ":8080"
	log.Println("serving on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
