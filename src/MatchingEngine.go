package main

import (
	"./tcp_server"
	log "github.com/sirupsen/logrus"
	"os"
	)

func init() {
	// Log as JSON instead of the default ASCII formatter.
  //log.SetFormatter(&log.JSONFormatter{})

  // Output to stdout instead of the default stderr
  // Can be any io.Writer, see below for File example
  log.SetOutput(os.Stdout)

  // Only log the warning severity or above.
  //log.SetLevel(log.WarnLevel)
}

func main() {
	var clientCount = 0

	server := tcp_server.New("localhost:12345")

	server.OnNewClient(func(c *tcp_server.Client) {
		// New Client Connected
		log.WithFields(log.Fields{
    "client count":  clientCount,
  }).Info("New client connection")
		clientCount += 1

		c.Send("Hello")

	})
	server.OnNewMessage(func(c *tcp_server.Client, message string) {
		// New Message Received
		log.Info("New message received")
	})
	server.OnClientConnectionClosed(func(c *tcp_server.Client, err error) {
		// Lost connection with Client
		log.Info("New client connection")

	})

	server.Listen()
}
