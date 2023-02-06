package model

import (
	"github.com/TUM-Dev/meldeplattform/pkg/i18n"
	"html/template"
)

type Config struct {
	Port    int    `yaml:"port"`
	Mode    string `yaml:"mode"`
	Content struct {
		Title    i18n.Translatable `yaml:"title"`
		SubTitle i18n.Translatable `yaml:"subtitle"`
		Logo     string            `yaml:"logo"`
		Topics   []Topic           `yaml:"topics"`
	} `yaml:"content"`
	HTTPS struct {
		Port int    `yaml:"port"`
		Cert string `yaml:"cert"`
		Key  string `yaml:"key"`
	} `yaml:"https"`
	FileDir string `yaml:"fileDir"`
	URL     string `yaml:"URL"`
	Saml    struct {
		IdpMetadataURL string `yaml:"idpMetadataURL"`
		EntityID       string `yaml:"entityID"`
		RootURL        string `yaml:"rootURL"`
		Cert           struct {
			Org           string `yaml:"org"`
			Country       string `yaml:"country"`
			Province      string `yaml:"province"`
			Locality      string `yaml:"locality"`
			StreetAddress string `yaml:"streetAddress"`
			PostalCode    string `yaml:"postalCode"`
			Cn            string `yaml:"cn"`
		} `yaml:"cert"`
	} `yaml:"saml"`
}

type Topic struct {
	Name    i18n.Translatable `yaml:"name"`
	Summary i18n.Translatable `yaml:"summary"`
	Fields  []struct {
		Name        i18n.Translatable `yaml:"name"`
		Type        string            `yaml:"type"` // e.g. file, text, email, textarea,
		Required    bool              `yaml:"required"`
		Description i18n.Translatable `yaml:"description"`

		// For select inputs:
		Choices *[]string `yaml:"choices"`
	} `yaml:"fields"`
	Contacts struct {
		Email   *EmailConfig   `yaml:"email"`
		Matrix  *MatrixConfig  `yaml:"matrix"`
		Webhook *WebhookConfig `yaml:"webhook"`
	} `yaml:"contacts"`
}

func (c Config) GetLogo() template.HTML {
	return template.HTML(c.Content.Logo)
}
