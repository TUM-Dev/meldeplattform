package model

import (
	"github.com/TUM-Dev/meldeplattform/pkg/i18n"
	"html/template"
)

type Config struct {
	Port    int    `yaml:"port"`
	Mode    string `yaml:"mode"`
	Content struct {
		Title      i18n.Translatable `yaml:"title"`
		SubTitle   i18n.Translatable `yaml:"subtitle"`
		Logo       string            `yaml:"logo"`
		AdminUsers []string          `yaml:"adminUsers"` // e.g. ge42tum
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
	Mail struct {
		User       string `yaml:"user"`
		Password   string `yaml:"password"`
		SMTPServer string `yaml:"smtpServer"`
		SMTPPort   string `yaml:"smtpPort"`
		From       string `yaml:"from"`
		FromName   string `yaml:"fromName"`
	} `yaml:"mail"`
}

func (c Config) GetLogo() template.HTML {
	return template.HTML(c.Content.Logo)
}
