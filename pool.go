package main

import "log"

type Pool struct {
	Register   chan *Client
	Unregister chan *Client
	Clients    map[*Client]bool
	Broadcast  chan Message
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan Message),
	}
}

func (pool *Pool) Start() {
	for {
		select {
		case client := <-pool.Register:
			pool.Clients[client] = true

			log.Println("New User Joined.")
			log.Println("Size of Connection Pool: ", len(pool.Clients))

			break

		case client := <-pool.Unregister:
			delete(pool.Clients, client)

			log.Println("User Disconnected.")
			log.Println("Size of Connection Pool: ", len(pool.Clients))

			break

		case message := <-pool.Broadcast:
			log.Println("Sending message to all clients in Pool")

			for client, _ := range pool.Clients {
				if err := client.Conn.WriteJSON(message); err != nil {
					log.Fatal(err)
					return
				}
			}
		}
	}
}
