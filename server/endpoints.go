package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"server/discord"
	"time"
)

func chatboxEndpoint(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		serveChatboxTemplate(w)
	case http.MethodPost:
		handleChatSubmit(w, r)
		http.Redirect(w, r, "/chatbox", http.StatusSeeOther)
	default:
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
	}
}

func statusEndpoint(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
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
	case http.MethodPost:
		user, pass, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "bad authorization", http.StatusUnauthorized)
			return
		}
		if user != "user" || pass != auth {
			http.Error(w, "bad authorization", http.StatusUnauthorized)
			return
		}

		decoder := json.NewDecoder(r.Body)
		var t StatusRequest
		err := decoder.Decode(&t)
		if err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		stati = append(stati, Status{Message: t.Message, Timestamp: time.Now()})
	default:
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return
	}
}

func statusJSONEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Add("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		err := encoder.Encode(stati)
		if err != nil {
			http.Error(w, `{"message": "something went wrong"}`, http.StatusInternalServerError)
			return
		}
	}
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	tmpl, err := template.ParseFiles("./templates/404.html")
	if err != nil {
		log.Fatal("error loading template: ", err)
		return
	}

	err = tmpl.Execute(w, r.URL.Path)
	if err != nil {
		log.Println("error rendering template: ", err)
	}
}

func messageEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		message := r.FormValue("message")

		if len(webhook) != 0 {
			err = discord.SendEmbed(webhook, discord.DiscordEmbed{
				Title:       "from: " + name,
				Description: message,
				Color:       0x458588,
			})
			if err != nil {
				log.Println("warning: failed to send message to webhook. check the URL")
				log.Println(err)
			}
		}

		tmpl, err := template.ParseFiles("./templates/sent.html")
		if err != nil {
			log.Fatal("error loading template: ", err)
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Println("error rendering template: ", err)
		}
	} else {
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return
	}
}

func chatEndpoint(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		liveChat(w, r)
	case http.MethodPost:
		handleChatSubmit(w, r)
	default:
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
	}
}
