package internal

import (
	"fmt"
	"github.com/joschahenningsen/meldeplattform/pkg/messaging"
	"gopkg.in/yaml.v2"
	"html/template"
	"os"
)

type config struct {
	Port    int `yaml:"port"`
	Content struct {
		Title   string `yaml:"title"`
		Summary string `yaml:"summary"`
		Logo    string `yaml:"logo"`
		Fields  []struct {
			Name        string `yaml:"name"`
			Type        string `yaml:"type"` // e.g. file, text, email, textarea,
			Required    bool   `yaml:"required"`
			Description string `yaml:"description"`

			// For select inputs:
			Choices *[]string `yaml:"choices"`
		} `yaml:"fields"`
	} `yaml:"content"`
	HTTPS struct {
		Port int    `yaml:"port"`
		Cert string `yaml:"cert"`
		Key  string `yaml:"key"`
	} `yaml:"https"`
	Forward struct {
		Email   *messaging.EmailConfig   `yaml:"email"`
		Matrix  *messaging.MatrixConfig  `yaml:"matrix"`
		Webhook *messaging.WebhookConfig `yaml:"webhook"`
	} `yaml:"forward"`
	FileDir string `yaml:"fileDir"`
	URL     string `yaml:"URL"`
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

func (c config) getMessengers() []messaging.Messenger {
	var messengers []messaging.Messenger

	if c.Forward.Email != nil {
		messengers = append(messengers, messaging.NewEmailMessenger(*c.Forward.Email))
	}
	if c.Forward.Matrix != nil {
		messengers = append(messengers, messaging.NewMatrixMessenger(*c.Forward.Matrix))
	}
	if c.Forward.Webhook != nil {
		messengers = append(messengers, messaging.NewWebhookMessenger(*c.Forward.Webhook))
	}

	return messengers
}

func (c config) GetLogo() template.HTML {
	return template.HTML(c.Content.Logo)
}
