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

func sendEmbed(webhookURL string, embed DiscordWebhook) error {
	jsonPayload, _ := json.Marshal(embed)
	req, _ := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
