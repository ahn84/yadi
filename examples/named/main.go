package main

import (
	"fmt"

	"github.com/ahn84/yadi"
)

type Messenger interface {
	Send(message string)
}

type EmailMessenger struct{}

func (m *EmailMessenger) Send(message string) {
	fmt.Printf("Sending email: %s\n", message)
}

type SmsMessenger struct{}

func (m *SmsMessenger) Send(message string) {
	fmt.Printf("Sending SMS: %s\n", message)
}

func main() {
	di.BindNamed("email", func() Messenger {
		return &EmailMessenger{}
	})
	di.BindNamed("sms", func() Messenger {
		return &SmsMessenger{}
	})

	var emailMessenger Messenger
	di.ResolveNamed(&emailMessenger, "email")
	emailMessenger.Send("Hello from email!")

	var smsMessenger Messenger
	di.ResolveNamed(&smsMessenger, "sms")
	smsMessenger.Send("Hello from SMS!")
}
