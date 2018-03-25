package main

import (
	"./tcp_server"
	"github.com/golang/glog"
	)

func main() {
	server := tcp_server.New("127.0.0.1:12345")

	server.OnNewClient(func(c *tcp_server.Client) {
		// New Client Connected
		glog.Info("New client connection")
		c.Send("Hello")
	})
	server.OnNewMessage(func(c *tcp_server.Client, message string) {
		// New Message Received
		glog.Info("New message received")
	})
	server.OnClientConnectionClosed(func(c *tcp_server.Client, err error) {
		// Lost connection with Client
		glog.Info("New client connection")
	})

	server.Listen()
}
