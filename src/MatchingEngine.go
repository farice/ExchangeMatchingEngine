package main

import (
	"./tcp_server"
	log "github.com/sirupsen/logrus"
	"os"
	"fmt"
	)

type StdOutHook struct{}

func (h *StdOutHook) Levels() []log.Level {
		return []log.Level{
			log.InfoLevel,
			log.WarnLevel,
			log.ErrorLevel,
			log.FatalLevel,
			log.PanicLevel,
		}
	}

	var fmter = new(log.TextFormatter)

	func (h *StdOutHook) Fire(entry *log.Entry) (err error) {
		line, err := fmter.Format(entry)
		if err == nil {
			fmt.Fprintf(os.Stderr, string(line))
		}
		return
	}

func init() {
	// Log as JSON instead of the default ASCII formatter.
  //log.SetFormatter(&log.JSONFormatter{})

	// You could set this to any `io.Writer` such as a file
   file, err := os.OpenFile("logs/exchange.log", os.O_CREATE|os.O_WRONLY, 0666)
   if err == nil {
    log.SetOutput(file)
   } else {
    log.Info("Failed to log to file, using default stdout")
   }

	 log.AddHook(&StdOutHook{})

  // Only log the warning severity or above.
  //log.SetLevel(log.WarnLevel)
}

func main() {
	var clientCount = 0

	server := tcp_server.New("exchange:12345")

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
		log.WithFields(log.Fields{
    "message": message,
  }).Info("New message received")
	})
	server.OnClientConnectionClosed(func(c *tcp_server.Client, err error) {
		// Lost connection with Client
		log.WithFields(log.Fields{
    "error": err,
  }).Info("Connection closed")

	})

	server.Listen()
}
