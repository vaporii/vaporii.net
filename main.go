package main

import (
	"crypto/hmac"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/big"
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
	clients   = make(map[string]chan Message) // map user ids to channel
	clientsMu sync.Mutex
	users     = make(map[string]*User) // map user ids to users
	usersMu   sync.Mutex
)

var colors = [...]string{"#CC241D", "#98971A", "#D79921", "#458588", "#B16286", "#689D6A", "#D65D0E", "#FB4934", "#B8BB26", "#FABD2F", "#83A598", "#D3869B", "#8EC07C", "#FE8019"}

type User struct {
	Color    string
	UserID   string
	Username string
}

type Message struct {
	UserID  string
	Message string
}

type ClientMessage struct {
	Username string
	Color    string
	Message  string
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	result := make([]byte, length)

	for i := range result {
		n, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			log.Fatal("random string failed???")
			return ""
		}
		result[i] = charset[n.Int64()]
	}

	return string(result)
}

func broadcast(msg Message) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	for _, ch := range clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func sendToUserID(userID string, msg Message) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if ch, exists := clients[userID]; exists {
		select {
		case ch <- msg:
		default:
		}
	}
}

func signUserID(username string) string {
	sig := computeSignature(username)
	return username + "|" + sig
}

func verifyUserID(signedValue string) (string, bool) {
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

func transformMessage(message Message) (ClientMessage, error) {
	user, ok := users[message.UserID]
	if !ok {
		return ClientMessage{
			Username: "error",
			Color:    "#FF0000",
			Message:  "user not found",
		}, errors.New("user not found")
	}

	return ClientMessage{
		Username: user.Username,
		Color:    user.Color,
		Message:  message.Message,
	}, nil
}

func liveChat(w http.ResponseWriter, r *http.Request) {
	// this doesn't work without charset=utf-8 for some reason (dont ask)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	cookie, err := getCookie(w, r)
	if err != nil {
		http.Error(w, "don't know what went wrong", http.StatusInternalServerError)
		return
	}

	userID, valid := verifyUserID(cookie.Value)
	if !valid {
		http.Error(w, "invalid username sig", http.StatusUnauthorized)
		return
	}

	clientChan := make(chan Message, 10)

	clientsMu.Lock()
	clients[userID] = clientChan
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, userID)
		clientsMu.Unlock()
	}()

	sendLocalMessage(userID, "hi!! please just have basic human decency")
	sendLocalMessage(userID, "use /nick to change your username")

	fmt.Fprintf(w, `
	<!doctype html>
	<html class="width-full">
		<head>
			<meta charset="UTF-8" />
			<meta http-equiv="Content-type" content="text/html;charset=UTF-8">
			<link rel="stylesheet" href="/style.css" />
		</head>
		<body class="transparent-bg width-full">
			<div class="flex flex-column-reverse message-div width-full break-word muted">
				<div>`)
	w.(http.Flusher).Flush()

	tmpl, err := template.ParseFiles("./templates/message.html")
	if err != nil {
		log.Fatal("error loading template: ", err)
	}

	for {
		select {
		case msg := <-clientChan:
			clientMsg, err := transformMessage(msg)
			if err != nil {
				// http.Error(w, err.Error(), http.StatusBadRequest)
				continue
			}
			err = tmpl.Execute(w, clientMsg)
			if err != nil {
				log.Fatal("error rendering template: ", err)
				return
			}
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
		// fmt.Fprintf(w, "<p>%s</p>\r\n", randomString(10))
		// time.Sleep(time.Second)
	}
}

func getCookie(w http.ResponseWriter, r *http.Request) (*http.Cookie, error) {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		userID := randomString(20)
		signed := signUserID(userID)

		cookie = &http.Cookie{
			Name:     "user_id",
			Value:    signed,
			HttpOnly: true,
			Secure:   false,
			Path:     "/",
			MaxAge:   0,
		}

		http.SetCookie(w, cookie)
	}

	usernameCookie, err := r.Cookie("username")
	if err != nil {
		username := "guest " + randomString(4)
		signed := signUserID(username)

		usernameCookie = &http.Cookie{
			Name:     "username",
			Value:    signed,
			HttpOnly: true,
			Secure:   false,
			Path:     "/",
			MaxAge:   0,
		}

		http.SetCookie(w, usernameCookie)
	}

	rawUserID, valid := verifyUserID(cookie.Value)
	if !valid {
		return nil, errors.New("invalid user id cookie")
	}

	username, valid := verifyUserID(usernameCookie.Value)
	if !valid {
		return nil, errors.New("invalid username cookie")
	}

	// excellent
	// source of
	// vitamin c
	// same character length??

	usersMu.Lock()
	_, ok := users[rawUserID]
	if !ok {
		users[rawUserID] = &User{
			Color:    colors[rand.Intn(len(colors))],
			UserID:   rawUserID,
			Username: username,
		}
	}
	usersMu.Unlock()

	return cookie, nil
}

func sendLocalMessage(userID string, message string) {
	sendToUserID(userID, Message{
		UserID:  "local",
		Message: message,
	})
}

func handleChatSubmit(w http.ResponseWriter, r *http.Request) {
	cookie, err := getCookie(w, r)
	if err != nil {
		http.Error(w, "don't know what went wrong", http.StatusInternalServerError)
		return
	}

	userID, valid := verifyUserID(cookie.Value)
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
		sendLocalMessage(userID, "message too large")
		return
	}

	if len(message) == 0 {
		sendLocalMessage(userID, "message too small")
		return
	}

	if strings.HasPrefix(message, "/") {
		// implement better command handling
		args := strings.Split(message, " ")
		if len(args) == 0 {
			return
		}

		if args[0] == "/nick" {
			// user := users[userID]
			newUsername := strings.TrimSpace(strings.Join(args[1:], " "))
			if len(newUsername) > 16 {
				sendLocalMessage(userID, "username too long (must be 3-16 chars)")
				return
			}
			if len(newUsername) < 4 {
				sendLocalMessage(userID, "username too short (must be 3-16 chars)")
				return
			}

			users[userID].Username = newUsername

			sendLocalMessage(userID, "username set to "+newUsername)
		}

		return
	}

	broadcast(Message{
		Message: message,
		UserID:  userID,
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

func chatboxEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		serveChatboxTemplate(w)
	} else if r.Method == http.MethodPost {
		handleChatSubmit(w, r)
		http.Redirect(w, r, "/chatbox", http.StatusSeeOther)
	} else {
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
	}
}

func main() {
	users["local"] = &User{
		Color:    "#ebdbb2",
		UserID:   "local",
		Username: "local",
	}

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
	http.HandleFunc("/chatbox", chatboxEndpoint)

	port := ":8080"
	log.Println("serving on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
