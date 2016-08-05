package main

import (
	"strings"
	"sync"
)

// Channel xD
type Channel struct {
	Name         string
	mu           *sync.Mutex
	Messages     []string
	MessageCount int
	MessageIndex int
}

// NewChannel xD
func NewChannel(name string) *Channel {
	channel := Channel{}
	channel.Name = strings.ToLower(name)
	channel.MessageCount = 0
	channel.MessageIndex = 0
	channel.Messages = make([]string, 200)
	channel.mu = &sync.Mutex{}
	return &channel
}
