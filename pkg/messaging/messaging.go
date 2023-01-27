package messaging

import (
	"github.com/TUM-Dev/meldeplattform/pkg/model"
)

// Messenger is an interface for all messaging clients capable of sending a message
type Messenger interface {
	SendMessage(title string, message model.Message, reportURL string) error
}

// Config is a generic type for a messenger configuration
type Config interface {
}
