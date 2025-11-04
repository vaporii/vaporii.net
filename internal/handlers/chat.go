package handlers

import (
	"fmt"
	"log"
	"net/http"
	"server/internal/models"
	"strings"
	"text/template"
)

func ChatEndpoint(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		cookie, err := r.Cookie("token")

		liveChat(w, r)
	case http.MethodPost:
		handleChatSubmit(w, r)
	default:
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
	}
}

func liveChat(w http.ResponseWriter, r *http.Request) {
	user, err := getUser(w, r)
	if err != nil {
		http.Error(w, "don't know what went wrong", http.StatusInternalServerError)
		return
	}

	clientChan := make(chan models.Message, 10)

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
	}
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
