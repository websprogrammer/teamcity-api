package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

//  Websocket client
type Client struct {
	ID   string
	Conn *websocket.Conn
	Pool *Pool
}

func (c *Client) read() {
	defer func() {
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()
	data := &SocketData{}
	done := make(chan bool)

	var lastBuildId int64

	// Set handler for close messages received.
	c.Conn.SetCloseHandler(func(code int, text string) error {
		log.Println("Closing of Websocket connection.")
		done <- true
		close(done)
		return nil
	})

	// Call the connection ReadJSON method to receive messages.
	for {
		log.Printf("Message New Start: %s", data.Environment)

		if err := c.Conn.ReadJSON(&data); err != nil {

			// Add the check to prevent the closure of a closed channel.
			if !isChannelClosed(done) {
				log.Fatalf("Error reading json. %s", err)
			}

			break
		}

		send(c, data.Environment, &lastBuildId)

		go fetcher(done, c, data.Environment, &lastBuildId)
	}

}

// Handle the received information and send a message to the client if there are any updates.
func send(c *Client, env string, lastBuildId *int64) {
	value, err := redisClient.Get(env).Result()
	if err != nil {
		log.Fatalf("Error getting results from Redis. %s", err)
		return
	}

	var builds *Builds

	if err := json.Unmarshal([]byte(value), &builds); err != nil {
		log.Fatalf("Error unmarshaling json results. %s", err)
		return
	}

	if builds.Build[0].Id != *lastBuildId {
		if err := c.Conn.WriteMessage(websocket.TextMessage, []byte(value)); err != nil {
			log.Fatal(err)
			return
		}

		log.Println("New information has been updated.")
		*lastBuildId = builds.Build[0].Id
	}

}

func fetcher(done <-chan bool, c *Client, env string, lastBuildId *int64) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			send(c, env, lastBuildId)

		case <-done:
			log.Printf("Done %s", env)
			return
		}
	}
}
