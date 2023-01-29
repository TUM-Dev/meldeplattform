package model

// Messenger is an interface for all messaging clients capable of sending a message
type Messenger interface {
	SendMessage(title string, message Message, reportURL string) error
}

// MessagingConfig is a generic type for a messenger configuration
type MessagingConfig interface {
}

func GetMessengers(t Topic) []Messenger {
	var messengers []Messenger

	if t.Contacts.Email != nil {
		messengers = append(messengers, NewEmailMessenger(*t.Contacts.Email))
	}
	if t.Contacts.Matrix != nil {
		messengers = append(messengers, NewMatrixMessenger(*t.Contacts.Matrix))
	}
	if t.Contacts.Webhook != nil {
		messengers = append(messengers, NewWebhookMessenger(*t.Contacts.Webhook))
	}

	return messengers
}
