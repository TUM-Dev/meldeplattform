package model

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type WebhookMessenger struct {
	config WebhookConfig
}

type WebhookConfig struct {
	Target string `yaml:"target"`
}

func NewWebhookMessenger(config WebhookConfig) *WebhookMessenger {
	return &WebhookMessenger{config: config}
}

func (m *WebhookMessenger) SendMessage(title string, message Message, reportURL string) error {
	msg := map[string]string{"title": title, "message": string(message.GetBody())}
	marshal, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = http.Post(m.config.Target, "application/json", bytes.NewBuffer(marshal))
	return err
}
