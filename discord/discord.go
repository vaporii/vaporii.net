package discord

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type DiscordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

type DiscordWebhook struct {
	Embeds []DiscordEmbed `json:"embeds"`
}

func SendEmbed(webhookURL string, embed DiscordEmbed) error {
	jsonPayload, _ := json.Marshal(DiscordWebhook{
		Embeds: []DiscordEmbed{embed},
	})

	_, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))

	return err
}
