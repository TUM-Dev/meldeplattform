package messaging

import "fmt"

type EmailMessenger struct {
	config EmailConfig
}

type EmailConfig struct {
	Sender string `yaml:"sender"`
	Target string `yaml:"target"`
}

func NewEmailMessenger(config EmailConfig) *EmailMessenger {
	return &EmailMessenger{config: config}
}

func (m *EmailMessenger) SendMessage(title, message string) error {
	fmt.Println(title)
	fmt.Println(message)
	return nil
}
