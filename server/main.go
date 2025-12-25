package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var secretKey []byte
var auth string
var webhook string

var (
	clients   = make(map[string]chan Message) // map user ids to channel
	clientsMu sync.Mutex
	users     = make(map[string]*User) // map user ids to users
	usersMu   sync.Mutex
)

var stati = []Status{{Message: "starting up her server", Timestamp: time.Now()}}

func serveChatboxTemplate(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("./templates/chatbox.html")
	if err != nil {
		log.Fatal("error loading template: ", err)
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		log.Fatal("error rendering template: ", err)
		return
	}
	w.(http.Flusher).Flush()
}

func main() {
	users["local"] = &User{
		Color:    "#ebdbb2",
		UserID:   "local",
		Username: "local",
	}

	err := godotenv.Load()
	if err != nil {
		log.Println("couldn't load .env file:", err)
	}

	secret, present := os.LookupEnv("SECRET")
	if !present {
		log.Fatal("SECRET not present in .env, please see README.md")
		return
	}
	secretKey = []byte(secret)

	auth, present = os.LookupEnv("STATUS_AUTH")
	if !present {
		log.Fatal("STATUS_AUTH not present in .env, please see README.md")
		return
	}

	webhook, present = os.LookupEnv("DISCORD_WEBHOOK_URL")
	if !present {
		log.Println("warning: discord webhook isn't set up, you won't receive messages from the contact page, please see README.md")
		webhook = ""
	}

	dir := http.Dir("./public")
	fs := http.FileServer(dir)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}
		upath = path.Clean(upath)

		f, err := dir.Open(upath)
		if err != nil {
			if os.IsNotExist(err) {
				notFoundHandler(w, r)
				return
			}
		}

		if err == nil {
			f.Close()
		}

		fs.ServeHTTP(w, r)
	})

	http.HandleFunc("/chat", chatEndpoint)
	http.HandleFunc("/chatbox", chatboxEndpoint)
	http.HandleFunc("/status", statusEndpoint)
	http.HandleFunc("/status-json", statusJSONEndpoint)
	http.HandleFunc("/message", messageEndpoint)
	http.HandleFunc("/newyears.png", imageEndpoint)

	port := ":8080"
	log.Println("serving on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
