package mail

import (
	"crypto/tls"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"net"
	"net/mail"
	"net/smtp"
)

func SendMail(user, password, server, port, fromName, from, to, subject, body string) error {
	fromMail := mail.Address{Name: fromName, Address: from}
	toMail := mail.Address{Address: to}

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = fromMail.String()
	headers["To"] = toMail.String()
	headers["Subject"] = subject
	headers["Content-type"] = "text/html; charset=\"UTF-8\""

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	// sanitize body construct message
	p := bluemonday.UGCPolicy()
	sanitizedBody := p.Sanitize(body)
	message += "\r\n" + sanitizedBody

	// Connect to the SMTP Server
	servername := server + ":" + port
	host, _, _ := net.SplitHostPort(servername)
	auth := smtp.PlainAuth("", user, password, host)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	c, err := smtp.Dial(servername)
	if err != nil {
		return err
	}

	err = c.StartTLS(tlsconfig)
	if err != nil {
		return err
	}

	// Auth
	if err = c.Auth(auth); err != nil {
		return err
	}

	// To && From
	if err = c.Mail(fromMail.Address); err != nil {
		return err
	}

	if err = c.Rcpt(toMail.Address); err != nil {
		return err
	}

	// Data
	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return c.Quit()
}
