package messaging

import "fmt"

type WebhookMessenger struct {
	config WebhookConfig
}

type WebhookConfig struct {
	Target string `yaml:"target"`
}

func NewWebhookMessenger(config WebhookConfig) *WebhookMessenger {
	return &WebhookMessenger{config: config}
}

func (m *WebhookMessenger) SendMessage(title, message string) error {
	fmt.Println(title)
	fmt.Println(message)
	return nil
}
