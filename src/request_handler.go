package main

import (
	log "github.com/sirupsen/logrus"
	"encoding/xml"
	"bytes"
)

// Remember to capitalize field names so they are exported
type Account struct {
	XMLName xml.Name `xml:"account"`
		Id string `xml:"id,attr"`
		Balance string `xml:"balance,attr"`
}

type Symbol struct {
			XMLName xml.Name `xml:"symbol"`
			Sym string `xml:"sym,attr"`
			Accounts []struct {
				Id string `xml:"id,attr"`
				Amount string `xml:",innerxml"`
				} `xml:"account"`

			}

			func createAccount(acct *Account) {
				// This creates a new account with the given unique ID and balance (in USD).
				// The account has no positions. Attempting to create an account that already
				// exists is an error.
			}

			func createSymbol(acct *Account) {
				// This creates the specified symbol. The symbol tag can have one or more
				//children which are <account id="ID">NUM</account> These indicate that
				// NUM shares of the symbol being created should be placed into the account
				// with the given ID. Note that this creation is legal even if sym already
				// exists: in such a case, it is used to create more shares of that symbol
				//and add them to existing accounts.
			}

			func parseXML(req []byte) {
				decoder := xml.NewDecoder(bytes.NewReader(req))
				var inElement string
				for {
					// Read tokens from the XML document in a stream.
					token, _ := decoder.Token()
					if token == nil {
						break
					}
					// Inspect the type of the token just read.
					switch se := token.(type) {
					case xml.StartElement:
						// If we just read a StartElement token
						inElement = se.Name.Local
						// ...and its name is "create"
						if inElement == "create" {
							for {
								// now we look, in order, at which create operations the user requests...
								token_create, _ := decoder.Token()
								if token_create == nil {
									break
								}
								switch se := token_create.(type) {
								case xml.StartElement:
									inElement = se.Name.Local
									switch inElement {
									// symbol create
									case "symbol":
										var symb Symbol
										err := decoder.DecodeElement(&symb, &se)
										if err != nil {
											log.WithFields(log.Fields{
												"Error": err,
												}).Error("Decoding error, symbol")

											// TODO - Handle error
											break
										}
										log.WithFields(log.Fields{
											"XML": symb,
											}).Info("New create command: Symbol")
									// symbol create
									case "account":
										var acct Account
										err := decoder.DecodeElement(&acct, &se)
										if err != nil {
											log.WithFields(log.Fields{
												"Error": err,
												}).Error("Decoding error, symbol")

											// TODO - Handle error
											break
										}

										log.WithFields(log.Fields{
											"XML": acct,
											}).Info("New create command: Account")
									default:

									}
								case xml.EndElement:
									inElement = se.Name.Local
									// we've reached the end of this create chunk
									if inElement == "create" {
										break
									}
								default:
								}
							}
						}
					default:
					}
				}
			}

			// Send bytes to Connection
			func (c *Connection) handleRequest(req []byte) {
				// New Message Received
				parseXML(req)

				}
