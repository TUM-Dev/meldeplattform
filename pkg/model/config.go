package model

import (
	"github.com/TUM-Dev/meldeplattform/pkg/i18n"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
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
	Imprint string `yaml:"imprint"`
	Privacy string `yaml:"privacy"`
}

func (c Config) GetPrivacy() template.HTML {
	return c.toHtml(c.Privacy)
}

func (c Config) GetImprint() template.HTML {
	return c.toHtml(c.Imprint)
}

func (c Config) toHtml(s string) template.HTML {
	unsafeHTML := blackfriday.Run([]byte(s), blackfriday.WithExtensions(blackfriday.CommonExtensions))
	// Sanitize the HTML to prevent XSS
	p := bluemonday.UGCPolicy()
	safeHTML := p.SanitizeBytes(unsafeHTML)
	return template.HTML(safeHTML)
}

func (c Config) GetLogo() template.HTML {
	return c.toHtml(c.Content.Logo)
}
