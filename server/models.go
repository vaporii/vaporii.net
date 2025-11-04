package main

import "time"

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

type StatusRequest struct {
	Message string `json:"message"`
}

type Status struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
