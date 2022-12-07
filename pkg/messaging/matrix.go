package messaging

import "fmt"

type MatrixMessenger struct {
	config MatrixConfig
}

type MatrixConfig struct {
	AccessToken string `yaml:"accessToken"`
	RoomID      string `yaml:"roomID"`
	HomeServer  string `yaml:"homeServer"`
}

func NewMatrixMessenger(config MatrixConfig) *MatrixMessenger {
	return &MatrixMessenger{config: config}
}

func (m *MatrixMessenger) SendMessage(message string) error {
	fmt.Println(message)
	return nil
}
