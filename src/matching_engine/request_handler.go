package main

import (
	log "github.com/sirupsen/logrus"
	"encoding/xml"
	"bytes"
	"github.com/farice/EME/redis"
	"strconv"
	"fmt"
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

	type CreatedResponse struct {
		XMLName xml.Name `xml:"created"`
		Sym string `xml:"sym,attr,omitempty"`
		Id string `xml:"id,attr,omitempty"`
	}

	type ErrorResponse struct {
		XMLName xml.Name `xml:"error"`
		Sym string `xml:"sym,attr,omitempty"`
		Id string `xml:"id,attr,omitempty"`
		Reason string `xml:",innerxml"`
	}

	func createAccount(acct *Account) (error) {
		// This creates a new account with the given unique ID and balance (in USD).
		// The account has no positions. Attempting to create an account that already
		// exists is an error.
		ex, _ := redis.Exists("acct:" + acct.Id)
		if acct.Id == "" {
			// TODO: - Throw and handle error
			return nil
		}
		if ex {
			log.WithFields(log.Fields{
				"ID": acct.Id,
				}).Info("Duplicate account")
			return fmt.Errorf("Duplicate account")
		}

		// Redis HMSET, maps key to hashmap of fields to values
		err := redis.SetField("acct:" + acct.Id, "balance", acct.Balance)

		if err != nil {
			log.WithFields(log.Fields{
				"Error": err,
				}).Error("error setting account")
				return fmt.Errorf("Error creating account")
		}

		// TEST: - Retrieve key + field, then log
		bal , _ := redis.GetField("acct:" + acct.Id, "balance")
		bal_float, _ := strconv.ParseFloat(string(bal.([]byte)), 64)
		log.WithFields(log.Fields{
			"ID": acct.Id,
			"Balance": bal_float,
			"Verify_Balance": acct.Balance,
			}).Info("Created account")
		// END TEST

		return nil
	}

	func createSymbol(sym *Symbol) {
		// This creates the specified symbol. The symbol tag can have one or more
		//children which are <account id="ID">NUM</account> These indicate that
		// NUM shares of the symbol being created should be placed into the account
		// with the given ID. Note that this creation is legal even if sym already
		// exists: in such a case, it is used to create more shares of that symbol
		//and add them to existing accounts.
		ex, _ := redis.Exists("sym:" + sym.Sym)
		if !ex {
			redis.Set("sym:" + sym.Sym, "")
		}

		for _, rcv_acct := range sym.Accounts {
			ex, _ := redis.Exists("acct:" + rcv_acct.Id)
			if !ex {
				// TODO:- Handle error, account does not exist
				return
			} else {
				// acct:ID:positions is a hashmap of all of the user's positions
				ex, _ := redis.HExists("acct:" + rcv_acct.Id + ":positions", sym.Sym)

				if ex {
					amt_float, _ := strconv.ParseFloat(rcv_acct.Amount, 64)
					redis.HIncrByFloat("acct:" + rcv_acct.Id + ":positions", sym.Sym, amt_float)
				} else {
					redis.SetField("acct:" + rcv_acct.Id + ":positions", sym.Sym, rcv_acct.Amount)
				}

			}

			// TEST: - Retrieve key + field, then log
			bal , _ := redis.GetField("acct:" + rcv_acct.Id + ":positions", sym.Sym)
			bal_float, _ := strconv.ParseFloat(string(bal.([]byte)), 64)
			log.WithFields(log.Fields{
				"ID": rcv_acct.Id,
				"Position": bal_float,
				"Symbol": sym.Sym,
				}).Info("Added shares to account")
			// END TEST

		}
    // element is the element from someSlice for where we are
}

	func parseXML(req []byte) (results string) {
		results += "<results>\n"
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

										createSymbol(&symb)

										// account create
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

											err = createAccount(&acct)
											if err == nil {
												succ := CreatedResponse{Id: acct.Id}
												if succ_string, err := xml.MarshalIndent(succ, "", "    "); err == nil {
													results += string(succ_string)
												}
											} else {
												fail := ErrorResponse{Id: acct.Id, Reason: err.Error()}
												if fail_string, err := xml.MarshalIndent(fail, "", "    "); err == nil {
													results += string(fail_string) + "\n"
												}
											}

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

							if inElement == "transactions" {
								// TODO: - transactions case
							}
							default:
							}
						}
						results += "</results>"
						return results
					}

					// Send bytes to Connection
					func (c *Connection) handleRequest(req []byte) {
						// New Message Received
						results := parseXML(req)
						c.Send(results)
					}
