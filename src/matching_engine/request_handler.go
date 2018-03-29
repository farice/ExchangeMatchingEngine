package main

import (
	log "github.com/sirupsen/logrus"
	"encoding/xml"
	"bytes"
	"github.com/farice/EME/redis"
	"strconv"
	"fmt"
	"sync"
)

var (
	mux sync.Mutex
)

// Inc increments the counter for the given key.
func IncAndGet ()(int) {
	mux.Lock()
	// Lock so only one goroutine at a time can access c.count
	ct, _ := redis.Incr("TransactionCounter")
	defer mux.Unlock()
	return ct
}

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

	type Order struct {
		XMLName xml.Name `xml:"order"`
		Sym string `xml:"sym,attr"`
		Amount string `xml:"amount,attr"` // negative means to sell
		Limit string `xml:"limit,attr"`
	}

	type Cancel struct {
		XMLName xml.Name `xml:"cancel"`
		TransactionID string `xml:"id,attr"`
	}

	type Query struct {
		XMLName xml.Name `xml:"query"`
		TransactionID string `xml:"id,attr"`
	}

	type OpenResponse struct {
		XMLName xml.Name `xml:"opened"`
		TransactionID string `xml:"id,attr"`
		Sym string `xml:"sym,attr"`
		Amount string `xml:"amount,attr"` // negative means to sell
		Limit string `xml:"limit,attr"`
	}

	type ErrorTransResponse struct {
		XMLName xml.Name `xml:"error"`
		Sym string `xml:"sym,attr"`
		Amount string `xml:"amount,attr"` // negative means to sell
		Limit string `xml:"limit,attr"`
		Reason string `xml:",innerxml"`
	}

	type CreatedResponse struct {
		XMLName xml.Name `xml:"created"`
		Sym string `xml:"sym,attr,omitempty"`
		Id string `xml:"id,attr,omitempty"`
	}

	type ErrorCreateResponse struct {
		XMLName xml.Name `xml:"error"`
		Sym string `xml:"sym,attr,omitempty"`
		Id string `xml:"id,attr,omitempty"`
		Reason string `xml:",innerxml"`
	}

	func createOrder(acctId string, order *Order) (transId int, err error) {
		log.Info("Creating order...")
		transId = IncAndGet()
		transId_str := strconv.Itoa(transId)
		limit_f, _ := strconv.ParseFloat(order.Limit, 64)
		if limit_f > 0 {
			err := redis.Zadd("open-buy:" + order.Sym, order.Limit, transId_str)
			if err != nil {
				// TODO - Error
			}
			} else {
				err := redis.Zadd("open-sell:" + order.Sym, order.Limit, transId_str)
				if err != nil {
					// TODO - Error
				}
			}
			return
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

					func createSymbol(sym *Symbol) (error){
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
								log.WithFields(log.Fields{
									"ID": rcv_acct.Id,
									}).Error("Account with ID does not exist")
									return fmt.Errorf("Account with ID %s does not exist", rcv_acct.Id)

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

										return nil
										// element is the element from someSlice for where we are
									}

									func parseXML(req []byte) (results string) {

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
													results += "<results>\n"
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

																		err = createSymbol(&symb)
																		if err == nil {
																			succ := CreatedResponse{Sym: symb.Sym}
																			if succ_string, err := xml.MarshalIndent(succ, "", "    "); err == nil {
																				results += string(succ_string) + "\n"
																			}
																			} else {
																				fail := ErrorCreateResponse{Sym: symb.Sym, Reason: err.Error()}
																				if fail_string, err := xml.MarshalIndent(fail, "", "    "); err == nil {
																					results += string(fail_string) + "\n"
																				}
																			}

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
																							results += string(succ_string) + "\n"
																						}
																						} else {
																							fail := ErrorCreateResponse{Id: acct.Id, Reason: err.Error()}
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

																			results += "</results>"
																			return results
																		}

																		if inElement == "transactions" {
																			// TODO - Search for key "id" rather than assuming that is it the only key
																			if se.Attr[0].Name.Local != "id" {
																				log.Error("Did not supply ID to perform transactions on")
																				break
																			}
																			trans_acct_id := se.Attr[0].Value

																			log.WithFields(log.Fields{
																				"Account ID": trans_acct_id ,
																				}).Info("Transactions on Account ID")

																				results += "<results>\n"
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
																						case "order":
																							var ord Order
																							err := decoder.DecodeElement(&ord, &se)
																							if err != nil {
																								log.WithFields(log.Fields{
																									"Error": err,
																									}).Error("Decoding error, order")

																									// TODO - Handle error
																									break
																								}

																								log.WithFields(log.Fields{
																									"parsed": ord,
																									}).Info("Order")

																									tr_id, err := createOrder(trans_acct_id, &ord)
																									if err == nil {
																										succ := OpenResponse{TransactionID: strconv.Itoa(tr_id), Sym: ord.Sym, Amount: ord.Amount, Limit: ord.Limit}
																										if succ_string, err := xml.MarshalIndent(succ, "", "    "); err == nil {
																											results += string(succ_string) + "\n"
																										}
																										} else {
																											fail := ErrorTransResponse{Sym: ord.Sym, Amount: ord.Amount, Limit: ord.Limit, Reason: err.Error()}
																											if fail_string, err := xml.MarshalIndent(fail, "", "    "); err == nil {
																												results += string(fail_string) + "\n"
																											}
																										}

																									case "cancel":
																										var cancel Cancel
																										err := decoder.DecodeElement(&cancel, &se)
																										if err != nil {
																											log.WithFields(log.Fields{
																												"Error": err,
																												}).Error("Decoding error, cancel")

																												// TODO - Handle error
																												break
																											}

																											log.WithFields(log.Fields{
																												"parsed": cancel,
																												}).Info("Cancel")

																											case "query":
																												var qry Query
																												err := decoder.DecodeElement(&qry, &se)
																												if err != nil {
																													log.WithFields(log.Fields{
																														"Error": err,
																														}).Error("Decoding error, query")

																														// TODO - Handle error
																														break
																													}

																													log.WithFields(log.Fields{
																														"parsed": qry,
																														}).Info("Query")
																													default:

																													}
																												case xml.EndElement:
																													inElement = se.Name.Local
																													// we've reached the end of this create chunk
																													if inElement == "transactions" {
																														break
																													}
																												}

																											}
																											results += "</results>"
																											return results
																										}
																									default:
																									}
																								}
																								return ""
																							}

																							// Send bytes to Connection
																							func (c *Connection) handleRequest(req []byte) {
																								// New Message Received
																								results := parseXML(req)
																								c.Send(results)
																							}
