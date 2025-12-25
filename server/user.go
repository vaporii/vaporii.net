package main

import (
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fogleman/gg"
)

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
	}
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

func generateImage(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("assets/meme.png")
	if err != nil {
		http.Error(w, "failed to open image", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "failed to decode image", http.StatusInternalServerError)
		return
	}

	tzParam := r.URL.Query().Get("tz")
	loc := time.Local
	if tzParam != "" {
		loc, err = time.LoadLocation(tzParam)
		if err != nil {
			http.Error(w, "invalid timezone", http.StatusBadRequest)
			return
		}
	}

	now := time.Now().In(loc)
	nextYear := time.Date(now.Year()+1, time.January, 1, 0, 0, 0, 0, loc)
	duration := nextYear.Sub(now)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	// 804, 183

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	dc := gg.NewContextForRGBA(rgba)
	dc.SetColor(color.White)
	dc.DrawRectangle(784, 190, 172, 100)
	dc.Fill()

	dc.DrawRectangle(431, 316, 137, 100)
	dc.Fill()

	dc.DrawRectangle(400, 450, 137, 100)
	dc.Fill()

	dc.SetRGB(0, 0, 0)
	if err := dc.LoadFontFace("assets/Futura.ttf", 108); err != nil {
		log.Println("failed to load font")
	}
	dc.DrawStringAnchored(fmt.Sprintf("%02d", hours), 790, 269, 0, 0)
	dc.DrawStringAnchored(fmt.Sprintf("%02d", minutes), 437, 398, 0, 0)
	dc.DrawStringAnchored(fmt.Sprintf("%02d", seconds), 401, 529, 0, 0)

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Surrogate-Control", "no-store")

	if err := png.Encode(w, rgba); err != nil {
		log.Println("failed to encode image:", err)
	}
}
