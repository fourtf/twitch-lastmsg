package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const dateFormat = "20060102-150405"

var (
	channels     = make(map[string]*Channel)
	channelMutex = &sync.Mutex{}
	pongReceived = true
	_conn        net.Conn
	_writer      *bufio.Writer
	connected    = false
	settings     Settings
)

func main() {
	file, e := ioutil.ReadFile("./config.json")
	if e != nil {
		fmt.Printf("config.json not found: %v\n", e)
		os.Exit(1)
	}

	e = json.Unmarshal(file, &settings)
	if e != nil {
		fmt.Printf("error in config.json\n")
		os.Exit(1)
	}

	connect()

	for _, c := range settings.Channels {
		addChannel(c)
	}

	pingTimer := time.NewTicker(time.Second * 15)
	go func() {
		for range pingTimer.C {
			if connected {
				if !pongReceived {
					disconnect()
					connect()
				} else {
					_writer.WriteString("PING\n")
					_writer.Flush()
					fmt.Println("< PING")
				}
			}
		}
	}()

	http.HandleFunc("/lastmessages/", lastMessages)
	http.ListenAndServe(":"+strconv.Itoa(settings.HTTPServePort), nil)
}

func connect() {
	fmt.Println("< connect")
	conn, _ := net.Dial("tcp", "irc.chat.twitch.tv:6667")
	_conn = conn

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	_writer = writer

	channelMutex.Lock()
	for _, channel := range channels {
		_writer.WriteString("JOIN #" + channel.Name + "\n")
	}
	_writer.Flush()
	channelMutex.Unlock()

	connected = true

	writer.WriteString("NICK justinfan123\n")
	writer.Flush()
	fmt.Println("< NICK justinfan123")
	writer.WriteString("CAP REQ :twitch.tv/commands\n")
	writer.WriteString("CAP REQ :twitch.tv/tags\n")
	writer.Flush()

	go func() {
		for {
			text, err := reader.ReadString('\n')
			if err == nil {
				handleMessage(text)
				fmt.Print("> " + text)
			} else {
				break
			}
		}
	}()
}

func disconnect() {
	connected = false
	_conn.Close()
}

func addChannel(name string) {
	if connected {
		channelMutex.Lock()
		channels[name] = NewChannel(name)
		if connected {
			_writer.WriteString("JOIN #" + name + "\n")
			_writer.Flush()
		}
		channelMutex.Unlock()
	}
}

func handleMessage(msg string) {
	S := strings.Split(msg, " ")
	if len(S) > 0 && S[0] == "PONG" {
		pongReceived = true
	} else if len(S) > 3 && S[2] == "PRIVMSG" {
		channelName := S[3][1:]

		channelMutex.Lock()
		c, success := channels[channelName]
		channelMutex.Unlock()

		if success {
			_msg := "@timestamp-utc=" + time.Now().UTC().Format(dateFormat) + ";" + msg[1:]

			c.AddMessage(_msg)
			fmt.Printf("+ Added message for channel %s\n", c.Name)
		}
	}
}

func writeLastMessages(w http.ResponseWriter, c *Channel) {
	messages, messageCount, messageIndex := c.GetLastMessages()

	fmt.Printf("+ Writing %d last messages for %s\n", messageCount, c.Name)

	for index := 1; index <= messageCount; index++ {
		w.Write([]byte(messages[(messageIndex+index)%len(messages)]))
	}
}

func writeMessagesSince(w http.ResponseWriter, c *Channel, lastMessage *time.Time) {
	messages, messageCount, messageIndex := c.GetLastMessages()

	fmt.Printf("@ Writing messages since %s for %s\n", lastMessage, c.Name)

	for index := 1; index <= messageCount; index++ {
		msg := messages[(messageIndex+index)%len(messages)]
		// KKaper
		t, _ := time.Parse(dateFormat, msg[15:30])
		if lastMessage.Before(t) {
			w.Write([]byte(msg))
		}
	}
}

func lastMessages(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.String(), "/")
	fmt.Printf("%#v\n", parts)
	var channelName string
	if len(parts) < 3 {
		return
	}

	channelName = strings.ToLower(parts[2])

	channelMutex.Lock()
	c, success := channels[channelName]
	channelMutex.Unlock()

	if !success {
		w.Write([]byte("Channel does not exist"))
		return
	}

	writeLastMessages(w, c)
}

func lastMessagesWithTime(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.String(), "/")
	fmt.Printf("%#v\n", parts)
	var channelName string
	var lastMessage *time.Time
	if len(parts) < 3 {
		return
	}

	channelName = strings.ToLower(parts[2])
	if len(parts) >= 4 {
		t, err := time.Parse(dateFormat, parts[3])
		if err == nil {
			lastMessage = &t
		}
	}

	channelMutex.Lock()
	c, success := channels[channelName]
	channelMutex.Unlock()

	if !success {
		w.Write([]byte("Channel does not exist"))
		return
	}

	if lastMessage == nil {
		writeLastMessages(w, c)
	} else {
		writeMessagesSince(w, c, lastMessage)
	}
}
