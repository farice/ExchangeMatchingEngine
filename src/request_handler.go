package main

import (
	log "github.com/sirupsen/logrus"
)

// Send bytes to Connection
func (c *Connection) handleRequest(req []byte) {
  // New Message Received
  log.WithFields(log.Fields{
  "message": string(req),
}).Info("New message received")
}
