package messaging

// Messenger is an interface for all messaging clients capable of sending a message
type Messenger interface {
	SendMessage(m string) error
}

// Config is a generic type for a messenger configuration
type Config interface {
}
