package messaging

import (
	"github.com/TUM-Dev/meldeplattform/pkg/model"
	"log"
	"net/smtp"
	"strings"
)

type EmailMessenger struct {
	config EmailConfig
}

type EmailConfig struct {
	Relay  string `yaml:"relay"`
	Sender string `yaml:"sender"`
	Target string `yaml:"target"`
}

func NewEmailMessenger(config EmailConfig) *EmailMessenger {
	return &EmailMessenger{config: config}
}

func (m *EmailMessenger) SendMessage(title string, message model.Message, reportURL string) error {
	return m.sendMail(m.config.Relay, m.config.Sender, title, message.Content, []string{m.config.Target})
}

func (m *EmailMessenger) sendMail(addr, from, subject, body string, to []string) error {
	log.Printf("sending mail to %v, subject: %s body:\n%s", to, subject, body)
	r := strings.NewReplacer("\r\n", "", "\r", "", "\n", "", "%0a", "", "%0d", "")

	msg := "To: " + strings.Join(to, ",") + "\r\n" +
		"From: " + from + "\r\n" +
		"Subject: " + subject + "\r\n" +
		strings.ReplaceAll(body, "Content-Type: text/plain", "Content-Type: text/plain; charset=UTF-8")
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.Mail(r.Replace(from)); err != nil {
		return err
	}
	for i := range to {
		to[i] = r.Replace(to[i])
		if err = c.Rcpt(to[i]); err != nil {
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}
