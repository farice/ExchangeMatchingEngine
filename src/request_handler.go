package main

import (
	log "github.com/sirupsen/logrus"
  "encoding/xml"
)

// Remember to capitalize field names so they are exported 
type Create struct {
	XMLName xml.Name `xml:"create"`
  Accounts []struct {
    Id string `xml:"id,attr"`
    Balance string `xml:"balance,attr"`
  } `xml:"account"`

  Symbols []struct {
    Sym string `xml:"sym,attr"`
    Accounts []struct {
      Id string `xml:"id,attr"`
      Amount string `xml:",innerxml"`
    } `xml:"account"`

  } `xml:"symbol"`

}

// Send bytes to Connection
func (c *Connection) handleRequest(req []byte) {
  // New Message Received
  var createXML Create
  err := xml.Unmarshal(req, &createXML)

  if err != nil {
        log.Error("Error unmarshalling from XML", err)
        return
  }

  log.WithFields(log.Fields{
  "XML": createXML,
}).Info("New message received")

}
