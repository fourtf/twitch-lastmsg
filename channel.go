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

// AddMessage adds the given message to the current index of the Message slice
func (c *Channel) AddMessage(msg string) {
	c.mu.Lock()
	c.Messages[(c.MessageIndex+c.MessageCount+1)%len(c.Messages)] = msg
	if c.MessageCount == len(c.Messages) {
		c.MessageIndex++
	} else {
		c.MessageCount++
	}

	c.mu.Unlock()
}

/*
GetLastMessages returns the last messages written in the given channel.
Return values:
slice of messages in string format (including IRCv3 tags)
Message count
Start index
*/
func (c *Channel) GetLastMessages() ([]string, int, int) {
	c.mu.Lock()
	messages := make([]string, len(c.Messages))
	messageCount := c.MessageCount
	messageIndex := c.MessageIndex
	copy(messages, c.Messages)
	c.mu.Unlock()

	return messages, messageCount, messageIndex
}
