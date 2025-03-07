package main

import (
	"log"
	"net/http"
	// "html/template"
)

func main() {
	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

	// });

	fs := http.FileServer(http.Dir("./public"))

	http.Handle("/", http.StripPrefix("/", fs))

	port := ":8080"
	log.Println("serving on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
