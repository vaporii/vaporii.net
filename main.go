package main

import (
	cryptoRand "crypto/rand"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"server/since"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
var stati = [...]Status{{Message: "starting up her server", Timestamp: time.Now()}}

type Status struct {
	Message   string
	Timestamp time.Time
}

func (s Status) ConvertTime() string {
	return since.Since(s.Timestamp)
}

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

	user, err := getUser(w, r)
	if err != nil {
		http.Error(w, "don't know what went wrong", http.StatusInternalServerError)
		return
	}

	clientChan := make(chan Message, 10)

	clientsMu.Lock()
	clients[user.UserID] = clientChan
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, user.UserID)
		clientsMu.Unlock()
	}()

	sendLocalMessage(user.UserID, "hi!! please just have basic human decency")
	sendLocalMessage(user.UserID, "use /nick to change your username")

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

func getToken(tokenStr string) (*User, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}

	user := &User{
		Username: claims["username"].(string),
		Color:    claims["color"].(string),
		UserID:   claims["id"].(string),
	}

	return user, nil
}

func setToken(user *User, w http.ResponseWriter) error {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"color":    user.Color,
		"id":       user.UserID,
	})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return errors.New("failed to sign key: " + err.Error())
	}

	cookie := &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		HttpOnly: true,
		Secure:   false, // make this secure later
		Path:     "/",
		MaxAge:   0,
	}

	http.SetCookie(w, cookie)

	return nil
}

func getUser(w http.ResponseWriter, r *http.Request) (*User, error) {
	cookie, err := r.Cookie("token")
	var user *User
	if err != nil {
		user = &User{
			Color:    colors[rand.Intn(len(colors))],
			UserID:   "",
			Username: "guest_" + randomString(4),
		}

		err = setToken(user, w)
		if err != nil {
			return nil, err
		}
	} else {
		newUser, err := getToken(cookie.Value)
		if err != nil {
			return nil, err
		}
		user = newUser
	}

	// excellent
	// source of
	// vitamin c
	// same character length??

	usersMu.Lock()
	_, ok := users[user.UserID]
	if !ok {
		users[user.UserID] = user
	}
	usersMu.Unlock()

	return user, nil
}

func sendLocalMessage(userID string, message string) {
	sendToUserID(userID, Message{
		UserID:  "local",
		Message: message,
	})
}

func handleChatSubmit(w http.ResponseWriter, r *http.Request) {
	user, err := getUser(w, r)
	if err != nil {
		http.Error(w, "don't know what went wrong", http.StatusInternalServerError)
		return
	}

	err = r.ParseForm()
	if err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	message := r.FormValue("message")
	if len(message) > 200 {
		sendLocalMessage(user.UserID, "message too large")
		return
	}

	if len(message) == 0 {
		sendLocalMessage(user.UserID, "message too small")
		return
	}

	if strings.HasPrefix(message, "/") {
		// implement better command handling
		args := strings.Split(message, " ")
		if len(args) == 0 {
			return
		}

		if args[0] == "/nick" {
			newUsername := strings.TrimSpace(strings.Join(args[1:], " "))
			if len(newUsername) > 16 {
				sendLocalMessage(user.UserID, "username too long (must be 3-16 chars)")
				return
			}
			if len(newUsername) < 3 {
				sendLocalMessage(user.UserID, "username too short (must be 3-16 chars)")
				return
			}

			users[user.UserID].Username = newUsername
			err = setToken(users[user.UserID], w)
			if err != nil {
				sendLocalMessage(user.UserID, "error changing username")
				return
			}

			sendLocalMessage(user.UserID, "username set to "+newUsername)
		}

		return
	}

	broadcast(Message{
		Message: message,
		UserID:  user.UserID,
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

func statusEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.ParseFiles("./templates/status-page.gohtml")
	if err != nil {
		log.Fatal("error loading template: ", err)
		return
	}

	err = tmpl.Execute(w, stati)
	if err != nil {
		log.Println("error rendering template: ", err)
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
	http.HandleFunc("/status", statusEndpoint)

	port := ":8080"
	log.Println("serving on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))
}
