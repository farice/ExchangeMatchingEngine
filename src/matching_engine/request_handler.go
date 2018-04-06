package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	counter_mux sync.Mutex
	match_mux   sync.Mutex // Mutex used to atomically match/execute orders
)

// Inc increments the counter for the given key.
func IncAndGet() int {
	counter_mux.Lock()
	// Lock so only one goroutine at a time can access c.count
	ct, _ := SharedModel().incTransactionCounter()
	defer counter_mux.Unlock()
	return ct
}

// must call with lock held to perform atomically
func executeOrder(
	matchAtBuyPrice bool,
	// buy order info
	b_trId string, b_acctId string, b_sym string, b_limit string, b_amount string,
	// sell order info
	s_trId string, s_acctId string, s_sym string, s_limit string, s_amount string) (sharesToExecute float64, err error) {

	if b_sym != s_sym {
		err = fmt.Errorf("Symbol mismatch.")
		return
	}

	log.Info("Execute order")
	logAccount(b_acctId)
	logAccount(s_acctId)

	sym := b_sym
	var limit_usd float64
	if matchAtBuyPrice {
		limit_usd, _ = strconv.ParseFloat(b_limit, 64)
	} else {
		limit_usd, _ = strconv.ParseFloat(s_limit, 64)
	}
	s_amt_f, _ := strconv.ParseFloat(s_amount, 64)
	b_amt_f, _ := strconv.ParseFloat(b_amount, 64)
	sharesToExecute = math.Min(-1*s_amt_f, b_amt_f)

	log.WithFields(log.Fields{
		"sym":           sym,
		"matched_limit": limit_usd,
		"b_account_id":  b_acctId,
		"b_limit":       b_limit,
		"b_amount":      b_amount,
		"s_account_id":  s_acctId,
		"s_limit":       s_limit,
		"s_amount":      s_amount,
	}).Info("Matched open orders")

	// add shares to buyer's account (don't worry about seller, they had shares removed when order opened)
	SharedModel().addOrSetSharesToPosition(b_acctId, sym, sharesToExecute)
	// add money to seller's account
	err = SharedModel().addAccountBalance(s_acctId, sharesToExecute*limit_usd)
	if err != nil {
		return
	}

	// remove money from buyer because money wasn't removed yet at order open
	if !matchAtBuyPrice {
		err = SharedModel().addAccountBalance(b_acctId, -1*sharesToExecute*limit_usd)
		if err != nil {
			return
		}
	}

	exec_time := time.Now().String()
	err = SharedModel().updateSellOrderAmount(s_trId, s_amt_f+sharesToExecute)

	if err != nil {
		return
	}
	// Update in Executed shares list
	err = SharedModel().executedOrder(s_trId, -1*sharesToExecute, limit_usd, exec_time)
	if err != nil {
		return
	}
	err = SharedModel().updateBuyOrderAmount(b_trId, b_amt_f-sharesToExecute)
	if err != nil {
		return
	}
	err = SharedModel().executedOrder(b_trId, sharesToExecute, limit_usd, exec_time)

	if err != nil {
		return
	}

	if s_amt_f+sharesToExecute == 0 {
		err = SharedModel().closeOpenSellOrder(s_trId, sym)
	}

	if b_amt_f-sharesToExecute == 0 {
		err = SharedModel().closeOpenBuyOrder(b_trId, sym)
	}

	logAccount(b_acctId)
	logAccount(s_acctId)

	return

}

func (order *Order) handleBuy(acctId string, transId_str string, sym string, order_amt float64, limit_f float64) (err error) {
	log.Info("Handle buy")
	// check if user has enough USD in their account
	var bal_float float64
	// bal_float, err = getAccountBalance(acctId)
	bal_float, err = SharedModel().getAccountBalance(acctId)
	if err != nil {
		return
	}

	if order_amt*limit_f > bal_float {
		err = fmt.Errorf("Insufficient funds")
		return
	}

	log.WithFields(log.Fields{
		"transId":          transId_str,
		"buy amount (USD)": order_amt * limit_f,
		"balance":          bal_float,
	}).Info("Funds")

	err = SharedModel().createOrder(transId_str, acctId, sym, order.Limit, order.Amount, time.Now())

	if err != nil {
		return
	}
	// get open sell with lowest sell value
	var members []string

	println("ACQUIRING LOCK")

	match_mux.Lock()
	defer match_mux.Unlock() // in case exception is thrown, unlock when stack closes

	var amountUnexecuted = order_amt

	// loop until there are no more orders to execute
	for {
		members, err = SharedModel().getMinimumSellOrder(sym, limit_f)
		if err != nil {
			return
		}
		if len(members) > 0 {
			// get information on this matched order...
			// "account", "symbol", "limit", "amount"
			data, _ := SharedModel().getOrder(members[0])

			if len(data) != 5 {
				log.WithFields(log.Fields{
					"data":      data,
					"len(data)": len(data),
				}).Error("Corrupted data")

				err = fmt.Errorf("Corrupted data: matched order info")
				return
			}

			matched_limit_f, _ := strconv.ParseFloat(data[2], 64)
			if matched_limit_f < limit_f {
				var amountExecuted float64
				amountExecuted, err = executeOrder(false, transId_str, acctId, sym, order.Limit, order.Amount, members[0], data[0], data[1], data[2], data[3])
				amountUnexecuted -= amountExecuted
				if amountUnexecuted == 0 {
					break
				}

			} else {
				break
			}

		} else {
			break
		}
	}

	log.WithFields(log.Fields{
		"Amount Unexecuted": amountUnexecuted,
	}).Info("Status")

	if amountUnexecuted > 0 {
		// No matches, add to open buy sorted set
		SharedModel().createBuyOrder(transId_str, acctId, sym, amountUnexecuted, order.Limit, limit_f)
		if err != nil {
			return
		}

		// remove funds from account, because not all was matched immediately
		SharedModel().addAccountBalance(acctId, -1*amountUnexecuted*limit_f)
	}

	logAccount(acctId)
	return
}

func (order *Order) handleSell(acctId string, transId_str string, sym string, order_amt float64, limit_f float64) (err error) {
	log.Info("handle sell")
	// check if user has enough of SYM in their account
	so_float, err := SharedModel().getPositionAmount(acctId, sym)
	if err != nil {
		return
	}

	if -1*order_amt > so_float {
		err = fmt.Errorf("Insufficient funds")
		return
	}

	log.WithFields(log.Fields{
		"transId":      transId_str,
		"sell amount":  -1 * order_amt,
		"shares owned": so_float,
	}).Info("Holdings")
	logAccount(acctId)

	// set order details
	err = SharedModel().createOrder(transId_str, acctId, sym, order.Limit, order.Amount, time.Now())

	if err != nil {
		return
	}

	var members []string
	match_mux.Lock()
	defer match_mux.Unlock() // in case exception is thrown, unlock when stack closes

	// remove shares from user's account
	SharedModel().addSharesToPosition(acctId, sym, order_amt)

	var sharesRemaining = order_amt // shares left to sell (<= 0)

	for {

		// find highest open buy order
		members, err = SharedModel().getMaximumBuyOrder(sym, limit_f)
		if err != nil {
			return
		}

		if len(members) > 0 {
			// get information on this matched order...
			data, _ := SharedModel().getOrder(members[0])

			if len(data) != 5 {
				log.WithFields(log.Fields{
					"data":      data,
					"len(data)": len(data),
				}).Error("Corrupted data")
				err = fmt.Errorf("Corrupted data: matched order info")
				return
			}

			matched_limit_f, _ := strconv.ParseFloat(data[2], 64)
			// price is executable
			if matched_limit_f > limit_f {
				// <=0
				var amountExecuted float64
				amountExecuted, err = executeOrder(true, members[0], data[0], data[1], data[2], data[3], transId_str, acctId, sym, order.Limit, order.Amount)
				sharesRemaining += amountExecuted
				if sharesRemaining == 0 {
					break
				}
			} else {
				break
			}
		} else {
			break
		}
	}

	log.WithFields(log.Fields{
		"Amount Unexecuted": sharesRemaining,
	}).Info("Status")

	// more shares to sell, still
	if sharesRemaining < 0 {
		// No matches, add to open sell sorted set
		err = SharedModel().createSellOrder(transId_str, acctId, sym, sharesRemaining, order.Limit, limit_f)
		if err != nil {
			return
		}

	}
	logAccount(acctId)

	return
}

func getOrderStatus(trId string) (resp string, err error) {
	log.Info("Get order status")
	ex, _ := SharedModel().transactionExists(trId)
	if !ex {
		resp = ""
		err = fmt.Errorf("Transaction does not exist")
		return
	}

	// "account", "symbol", "limit", "amount", "origAmount"
	order_info, _ := SharedModel().getOrder(trId)
	log.WithFields(log.Fields{
		"order info: [acct, sym, lim, amt, o_amt]": order_info,
	}).Info("Status transaction")

	transactions, _ := getPartialExecutions(trId)
	log.WithFields(log.Fields{
		"Transactions": transactions,
	}).Info("Execution history")

	if len(transactions)%3 != 0 {
		resp = ""
		err = fmt.Errorf("Malformed Redis Data")
		return
	}

	len_trans := len(transactions) / 3
	for i := 0; i < len_trans; i++ {
		exec := ExecutedQueryResponse{Shares: transactions[3*i], Price: transactions[3*i+1], Time: transactions[3*i+2]}
		if exec_string, err := xml.MarshalIndent(exec, "", "    "); err == nil {
			resp += string(exec_string) + "\n"
		}
	}

	remaining_amount, _ := strconv.ParseFloat(order_info[3], 64)
	if remaining_amount != 0 {
		open := OpenQueryResponse{Shares: order_info[3]}
		if open_string, err := xml.MarshalIndent(open, "", "    "); err == nil {
			resp += string(open_string) + "\n"
		}
	} else { // may have been cancelled!
		ex, _ := SharedModel().orderCancelled(trId)
		if ex {
			cancel_info, _ := SharedModel().getCancelledOrderDetails(trId)
			cancel := CancelQueryResponse{Shares: cancel_info[0], Time: cancel_info[1]}
			if cancel_string, err := xml.MarshalIndent(cancel, "", "    "); err == nil {
				resp += string(cancel_string) + "\n"
			}
		}
	}

	logAccount(order_info[0])

	return
}

func (q *Query) handleQuery() (resp string, err error) {
	log.Info("handle query")
	resp += "<status>\n"

	trId := q.TransactionID
	if trId == "" {
		err = fmt.Errorf("Invalid Query")
		resp += "</status>"
		return
	}

	match_mux.Lock()
	defer match_mux.Unlock()

	status, err := getOrderStatus(trId)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Error")
		resp += "</status>"
		return
	}
	resp += status

	resp += "</status>"

	return
}

func (c *Cancel) handleCancel() (resp string, err error) {
	log.Info("handle cancel")
	trId := c.TransactionID
	if trId == "" {
		err = fmt.Errorf("Invalid Query")
		return
	}
	resp += "<canceled>\n"

	match_mux.Lock()
	defer match_mux.Unlock()

	ex, _ := SharedModel().transactionExists(trId)
	if !ex {
		resp = ""
		err = fmt.Errorf("Transaction does not exist")
		return
	}

	// "account", "symbol", "limit", "amount"
	data, err := SharedModel().getOrder(trId)
	acct, sym, limit, amt := data[0], data[1], data[2], data[3]
	if err != nil {
		return
	}

	log.WithFields(log.Fields{
		"order info: [acct, sym, lim, amt, o_amt]": data,
	}).Info("Cancelling order")
	logAccount(acct)

	if len(data) != 5 {
		err = fmt.Errorf("Malformed redis data")
		return
	}

	amt_f, _ := strconv.ParseFloat(amt, 64)
	buy := (amt_f > 0)

	if amt_f != 0 {

		// remove from open orders sorted set
		if buy {
			err = SharedModel().closeOpenBuyOrder(trId, sym)
		} else {
			err = SharedModel().closeOpenSellOrder(trId, sym)
		}

		if err != nil {
			return
		}

		if buy { // add money back to account if buy order
			limit_f, _ := strconv.ParseFloat(limit, 64)
			SharedModel().addAccountBalance(acct, limit_f*amt_f)

		} else { // add shares back to account if sell order
			SharedModel().addOrSetSharesToPosition(acct, sym, -1*amt_f)
		}

		// set remaining amount to 0
		if buy {
			err = SharedModel().updateBuyOrderAmount(trId, 0.0)
		} else {
			err = SharedModel().updateSellOrderAmount(trId, 0.0)
		}

		if err != nil {
			return
		}

		// store info
		exec_time := time.Now().String()
		err = SharedModel().cancelOrder(trId, amt_f, exec_time)
		if err != nil {
			return
		}
	}

	status, err := getOrderStatus(trId)
	if err != nil {
		return
	}
	resp += status

	resp += "</canceled>"

	logAccount(acct)
	return
}

func (order *Order) openOrder(acctId string) (transId int, err error) {
	log.Info("Open order")
	transId = IncAndGet()
	transId_str := strconv.Itoa(transId)
	sym := order.Sym
	order_amt, _ := strconv.ParseFloat(order.Amount, 64)
	limit_f, _ := strconv.ParseFloat(order.Limit, 64)

	// BUY
	if order_amt > 0 {

		err = order.handleBuy(acctId, transId_str, sym, order_amt, limit_f)

		// SELL
	} else {

		err = order.handleSell(acctId, transId_str, sym, order_amt, limit_f)

	}
	return
}

func (acct *Account) createAccount() (err error) {
	log.Info("Create account")
	err = SharedModel().createAccount(acct.Id, acct.Balance)
	return err
}

func createSymbol(sym *Symbol) (err error) {
	log.Info("Create symbol")
	// This creates the specified symbol. The symbol tag can have one or more
	//children which are <account id="ID">NUM</account> These indicate that
	// NUM shares of the symbol being created should be placed into the account
	// with the given ID. Note that this creation is legal even if sym already
	// exists: in such a case, it is used to create more shares of that symbol
	//and add them to existing accounts.
	SharedModel().createOrUpdateSymbol(sym.Sym)

	for _, rcv_acct := range sym.Accounts {
		ex, _ := SharedModel().accountExists(rcv_acct.Id)
		if !ex {
			// TODO:- Handle error, account does not exist
			log.WithFields(log.Fields{
				"ID": rcv_acct.Id,
			}).Error("Account with ID does not exist")
			return fmt.Errorf("Account with ID %s does not exist", rcv_acct.Id)

		} else {
			// acct:ID:positions is a hashmap of all of the user's positions
			amt_float, _ := strconv.ParseFloat(rcv_acct.Amount, 64)
			SharedModel().addOrSetSharesToPosition(rcv_acct.Id, sym.Sym, amt_float)

		}

		// TEST: - Retrieve key + field, then log
		bal_float, err := SharedModel().getPositionAmount(rcv_acct.Id, sym.Sym)
		if err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"ID":       rcv_acct.Id,
			"Position": bal_float,
			"Symbol":   sym.Sym,
		}).Info("Added shares to account")
		// END TEST

	}

	return
	// element is the element from someSlice for where we are
}

func parseXML(req []byte) (results string) {

	defer LogMethodTimeElapsed("request_handler.parseXML", time.Now())

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

							err = acct.createAccount()
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

				results += "</results>\n"
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
					"Account ID": trans_acct_id,
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

							tr_id, err := ord.openOrder(trans_acct_id)
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

							resp_c, _ := cancel.handleCancel()
							results += resp_c + "\n"

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

							resp_q, _ := qry.handleQuery()
							results += resp_q + "\n"
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
				results += "</results>\n"
				return results
			}

			if inElement == "dump" {
				outputDatabaseStateTruncated(50)
				// store the output to the var "results"
			}
		default:
		}
	}
	return ""
}

// Send bytes to Connection
func (c *Connection) handleRequest(req []byte) {
	// New Message Received
	defer LogMethodTimeElapsed("request_handler.handleRequest", time.Now())
	results := parseXML(req)
	c.Send(results)
}
