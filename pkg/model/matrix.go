package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

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

const matrixMSGApiURL = "https://%s/_matrix/client/r0/rooms/%s/send/m.room.message?access_token=%s"

func (m *MatrixMessenger) SendMessage(title string, message Message, reportURL string) error {
	msg := map[string]string{
		"msgtype":        "m.text",
		"format":         "org.matrix.custom.html",
		"formatted_body": "<h1>" + title + "</h1>" + string(message.GetBody()) + "<br><a href=\"" + reportURL + "\">View Report</a>",
		"body":           "# " + title + "\n\n" + message.Content + "\n\nView Report: " + reportURL}
	marshal, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	resp, err := http.Post(
		fmt.Sprintf(matrixMSGApiURL, m.config.HomeServer, m.config.RoomID, m.config.AccessToken),
		"application/json",
		bytes.NewBuffer(marshal),
	)
	r, err := io.ReadAll(resp.Body)
	log.Println(string(r), err)
	return err
}
