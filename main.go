package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	// "html/template"
)

var secretKey []byte

var (
	clients   = make(map[chan Message]bool)
	clientsMu sync.Mutex
)

type Message struct {
	Color    string
	Username string
	Message  string
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func broadcast(msg Message) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for ch := range clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func signUsername(username string) string {
	sig := computeSignature(username)
	return username + "|" + sig
}

func verifyUsername(signedValue string) (string, bool) {
	parts := strings.Split(signedValue, "|")
	if len(parts) != 2 {
		return "", false
	}
	username, providedSig := parts[0], parts[1]
	expectedSig := computeSignature(username)
	if !hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
		return "", false
	}

	return username, true
}

func computeSignature(username string) string {
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(username))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func liveChat(w http.ResponseWriter, r *http.Request) {
	// this doesn't work without charset=utf-8 for some reason (dont ask)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	// data := Message{
	// 	Message:  randomString(10),
	// 	Color:    "#D65D0E",
	// 	Username: "vaporii",
	// }
	clientChan := make(chan Message, 10)

	clientsMu.Lock()
	clients[clientChan] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, clientChan)
		clientsMu.Unlock()
	}()

	broadcast(Message{
		Color:    "#FFFFFF",
		Username: "admin",
		Message:  "someone joined!",
	})

	fmt.Fprintf(w, "<!doctype html><html><head><link rel='stylesheet' href='/style.css' /></head><body class='transparent-bg'>\r\n")
	w.(http.Flusher).Flush()

	tmpl, err := template.ParseFiles("./templates/message.html")
	if err != nil {
		log.Fatal("error loading template: ", err)
	}

	for {
		select {
		case msg := <-clientChan:
			err = tmpl.Execute(w, msg)
			if err != nil {
				log.Fatal("error rendering template: ", err)
			}
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
		// fmt.Fprintf(w, "<p>%s</p>\r\n", randomString(10))
		// time.Sleep(time.Second)
	}
}

func handleChatSubmit(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie("username")
	if err != nil {
		username := randomString(10)
		signed := signUsername(username)

		http.SetCookie(w, &http.Cookie{
			Name:     "username",
			Value:    signed,
			HttpOnly: true,
			Secure:   true,
			Path:     "/",
			MaxAge:   0,
		})
		return
	}
	cookie, err := r.Cookie("username")
	if err != nil {
		http.Error(w, "don't know what went wrong", http.StatusInternalServerError)
	}

	username, valid := verifyUsername(cookie.Value)
	if !valid {
		http.Error(w, "invalid username sig", http.StatusUnauthorized)
		return
	}

	err = r.ParseForm()
	if err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	message := r.FormValue("message")
	if len(message) > 200 {
		http.Error(w, "message too large", http.StatusBadRequest)
		return
	}

	broadcast(Message{
		Message:  message,
		Color:    "#689D6A",
		Username: username,
	})
}

func chatEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		liveChat(w, r)
	} else if r.Method == http.MethodPost {
		handleChatSubmit(w, r)
	} else {
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env file")
	}

	secret, present := os.LookupEnv("SECRET")
	if !present {
		log.Fatal("SECRET not present in .env")
	}
	secretKey = []byte(secret)

	fs := http.FileServer(http.Dir("./public"))

	http.Handle("/", http.StripPrefix("/", fs))
	http.HandleFunc("/chat", chatEndpoint)

	port := ":8080"
	log.Println("serving on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
