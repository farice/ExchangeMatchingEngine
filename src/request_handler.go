package main

import (
	log "github.com/sirupsen/logrus"
)

// Send bytes to Connection
func (c *Connection) handleRequest(req string) {
  // New Message Received
  log.WithFields(log.Fields{
  "message": req,
}).Info("New message received")
}
