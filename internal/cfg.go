package internal

import (
	"fmt"
	"github.com/TUM-Dev/meldeplattform/pkg/messaging"
	"gopkg.in/yaml.v2"
	"html/template"
	"os"
)

type config struct {
	Port    int    `yaml:"port"`
	Mode    string `yaml:"mode"`
	Content struct {
		Title  string  `yaml:"title"`
		Logo   string  `yaml:"logo"`
		Topics []topic `yaml:"topics"`
	} `yaml:"content"`
	HTTPS struct {
		Port int    `yaml:"port"`
		Cert string `yaml:"cert"`
		Key  string `yaml:"key"`
	} `yaml:"https"`
	FileDir string `yaml:"fileDir"`
	URL     string `yaml:"URL"`
}

type topic struct {
	Name    string `yaml:"name"`
	Summary string `yaml:"summary"`
	Fields  []struct {
		Name        string `yaml:"name"`
		Type        string `yaml:"type"` // e.g. file, text, email, textarea,
		Required    bool   `yaml:"required"`
		Description string `yaml:"description"`

		// For select inputs:
		Choices *[]string `yaml:"choices"`
	} `yaml:"fields"`
	Contacts struct {
		Email   *messaging.EmailConfig   `yaml:"email"`
		Matrix  *messaging.MatrixConfig  `yaml:"matrix"`
		Webhook *messaging.WebhookConfig `yaml:"webhook"`
	} `yaml:"contacts"`
}

func (a *App) initCfg() error {
	f, err := os.Open("config/config.yaml")
	if err != nil {
		return fmt.Errorf("open config.yaml: %v", err)
	}
	d := yaml.NewDecoder(f)
	if err = d.Decode(&a.config); err != nil {
		return fmt.Errorf("decode config.yaml: %v", err)
	}
	return nil
}

func (t topic) getMessengers() []messaging.Messenger {
	var messengers []messaging.Messenger

	if t.Contacts.Email != nil {
		messengers = append(messengers, messaging.NewEmailMessenger(*t.Contacts.Email))
	}
	if t.Contacts.Matrix != nil {
		messengers = append(messengers, messaging.NewMatrixMessenger(*t.Contacts.Matrix))
	}
	if t.Contacts.Webhook != nil {
		messengers = append(messengers, messaging.NewWebhookMessenger(*t.Contacts.Webhook))
	}

	return messengers
}

func (c config) GetLogo() template.HTML {
	return template.HTML(c.Content.Logo)
}
