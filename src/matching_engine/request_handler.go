package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/farice/EME/redis"
	redigo "github.com/gomodule/redigo/redis"
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
	ct, _ := redis.Incr("TransactionCounter")
	defer counter_mux.Unlock()
	return ct
}

// Remember to capitalize field names so they are exported

type Account struct {
	XMLName xml.Name `xml:"account"`
	Id      string   `xml:"id,attr"`
	Balance string   `xml:"balance,attr"`
}

type Symbol struct {
	XMLName  xml.Name `xml:"symbol"`
	Sym      string   `xml:"sym,attr"`
	Accounts []struct {
		Id     string `xml:"id,attr"`
		Amount string `xml:",innerxml"`
	} `xml:"account"`
}

type Order struct {
	XMLName xml.Name `xml:"order"`
	Sym     string   `xml:"sym,attr"`
	Amount  string   `xml:"amount,attr"` // negative means to sell
	Limit   string   `xml:"limit,attr"`
}

type Cancel struct {
	XMLName       xml.Name `xml:"cancel"`
	TransactionID string   `xml:"id,attr"`
}

type Query struct {
	XMLName       xml.Name `xml:"query"`
	TransactionID string   `xml:"id,attr"`
}

type OpenQueryResponse struct {
	XMLName xml.Name `xml:"open"`
	Shares string   `xml:"shares,attr"`
}

type CancelQueryResponse struct {
	XMLName xml.Name `xml:"canceled"`
	Shares string   `xml:"shares,attr"`
	Time string `xml:"time,attr"`
}

type ExecutedQueryResponse struct {
	XMLName xml.Name `xml:"executed"`
	Shares string   `xml:"shares,attr"`
	Price string   `xml:"price,attr"`
	Time string `xml:"time,attr"`
}

type OpenResponse struct {
	XMLName       xml.Name `xml:"opened"`
	TransactionID string   `xml:"id,attr"`
	Sym           string   `xml:"sym,attr"`
	Amount        string   `xml:"amount,attr"` // negative means to sell
	Limit         string   `xml:"limit,attr"`
}

type ErrorTransResponse struct {
	XMLName xml.Name `xml:"error"`
	Sym     string   `xml:"sym,attr"`
	Amount  string   `xml:"amount,attr"` // negative means to sell
	Limit   string   `xml:"limit,attr"`
	Reason  string   `xml:",innerxml"`
}

type CreatedResponse struct {
	XMLName xml.Name `xml:"created"`
	Sym     string   `xml:"sym,attr,omitempty"`
	Id      string   `xml:"id,attr,omitempty"`
}

type ErrorCreateResponse struct {
	XMLName xml.Name `xml:"error"`
	Sym     string   `xml:"sym,attr,omitempty"`
	Id      string   `xml:"id,attr,omitempty"`
	Reason  string   `xml:",innerxml"`
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
	addShares(b_acctId, sym, sharesToExecute)
	// add money to seller's account
	err = addAccountBalance(s_acctId, sharesToExecute*limit_usd)
	if err != nil {
		return
	}

	// remove money from buyer because money wasn't removed yet at order open
	if !matchAtBuyPrice {
		err = addAccountBalance(b_acctId, -1*sharesToExecute*limit_usd)
		if err != nil {
			return
		}
	}

	conn := redis.Pool.Get()
	exec_time := time.Now().String()
	_, err = conn.Do("HSET", "order:"+s_trId, "amount", s_amt_f+sharesToExecute)

	if err != nil {
		return
	}
	// Update in Executed shares list
	_, err = conn.Do("RPUSH", "order-executed:"+s_trId, -1 * sharesToExecute, limit_usd, exec_time)
	if err != nil {
		return
	}
	_, err = conn.Do("HSET", "order:"+b_trId, "amount", b_amt_f-sharesToExecute)
	if err != nil {
		return
	}
	_, err = conn.Do("RPUSH", "order-executed:"+b_trId, sharesToExecute, limit_usd, exec_time)

	if err != nil {
		return
	}
	conn.Close()

	if s_amt_f+sharesToExecute == 0 {
		closeOpenOrder(false, sym, s_trId)
	}

	if b_amt_f-sharesToExecute == 0 {
		closeOpenOrder(true, sym, b_trId)
	}

	return

}

func closeOpenOrder(buy bool, sym string, transId string) (err error) {
	conn := redis.Pool.Get()
	defer conn.Close()
	if buy {
		_, err = conn.Do("ZREM", "open-buy:"+sym, transId)
	} else {
		_, err = conn.Do("ZREM", "open-sell:"+sym, transId)
	}

	return

}

// func getAccountBalance(acctId string) (bal_f float64, err error) {
// 	bal, _ := redis.GetField("acct:"+acctId, "balance")
// 	if bal == nil {
// 		// If a user balance is nil, it does not exist
// 		err = fmt.Errorf("User %s does not exist", acctId)
// 		return
// 	}
// 	bal_f, _ = strconv.ParseFloat(string(bal.([]byte)), 64)

// 	return
// }

func addAccountBalance(acctId string, amount float64) (err error) {
	ex, _ := redis.HExists("acct:"+acctId, "balance")
	if ex == false {
		// If a user balance is nil, it does not exist
		err = fmt.Errorf("User %s does not exist", acctId)
		return
	}

	redis.HIncrByFloat("acct:"+acctId, "balance", amount)
	return
}

func addShares(acctId string, sym string, amount float64) {
	ex, _ := redis.HExists("acct:"+acctId+":positions", sym)

	if ex {

		redis.HIncrByFloat("acct:"+acctId+":positions", sym, amount)
	} else {
		redis.SetField("acct:"+acctId+":positions", sym, amount)
	}
}

func (order *Order) handleBuy(acctId string, transId_str string, sym string, order_amt float64, limit_f float64) (err error) {
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
		"transId": transId_str,
		"buy amount (USD)": order_amt * limit_f,
		"balance":          bal_float,
	}).Info("Funds")

	var conn = redis.Pool.Get()
	_, err = conn.Do("HMSET", "order:"+transId_str, "account", acctId, "symbol", sym, "limit", order.Limit, "amount", order.Amount, "origAmount", order.Amount)
	conn.Close()

	if err != nil {
		return
	}
	// get open sell with lowest sell value
	var members []string

	match_mux.Lock()
	defer match_mux.Unlock() // in case exception is thrown, unlock when stack closes
	conn = redis.Pool.Get()
	defer conn.Close()

	var amountUnexecuted = order_amt

	// loop until there are no more orders to execute
	for {
		members, err = redis.Zrange("open-sell:"+sym, 0, 0, true)
		if err != nil {
			return
		}
		if len(members) > 0 {
			// get information on this matched order...
			data, _ := redigo.Strings(conn.Do("HMGET", "order:"+members[0], "account", "symbol", "limit", "amount"))

			if len(data) != 4 {
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
		err = redis.Zadd("open-buy:"+sym, order.Limit, transId_str)
		if err != nil {
			return
		}

		// remove funds from account, because not all was matched immediately
		addAccountBalance(acctId, -1*amountUnexecuted*limit_f)
	}

	return
}

func (order *Order) handleSell(acctId string, transId_str string, sym string, order_amt float64, limit_f float64) (err error) {

	// check if user has enough of SYM in their account
	var shares_owned interface{}
	shares_owned, err = redis.GetField("acct:"+acctId+":positions", sym)
	if err != nil {
		return
	}
	if shares_owned == nil {
		err = fmt.Errorf("User owns no shares of %s", sym)
		return
	}
	so_float, _ := strconv.ParseFloat(string(shares_owned.([]byte)), 64)

	if -1*order_amt > so_float {
		err = fmt.Errorf("Insufficient funds")
		return
	}

	log.WithFields(log.Fields{
		"transId": transId_str,
		"sell amount":  -1 * order_amt,
		"shares owned": so_float,
	}).Info("Holdings")

	var conn = redis.Pool.Get()
	// set order details
	_, err = conn.Do("HMSET", "order:"+transId_str, "account", acctId, "symbol", sym, "limit", order.Limit, "amount", order.Amount, "origAmount", order.Amount)
	conn.Close()

	if err != nil {
		return
	}

	var members []string
	match_mux.Lock()
	defer match_mux.Unlock() // in case exception is thrown, unlock when stack closes
	conn = redis.Pool.Get()
	defer conn.Close()

	// remove shares from user's account
	redis.HIncrByFloat("acct:"+acctId+":positions", sym, order_amt)

	var sharesRemaining = order_amt // shares left to sell (<= 0)

	for {

		// find highest open buy order
		members, err = redis.Zrange("open-buy:"+sym, -1, -1, true)
		if err != nil {
			return
		}

		if len(members) > 0 {
			// get information on this matched order...
			data, _ := redigo.Strings(conn.Do("HMGET", "order:"+members[0], "account", "symbol", "limit", "amount"))

			if len(data) != 4 {
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
		err = redis.Zadd("open-sell:"+sym, order.Limit, transId_str)
		if err != nil {
			return
		}

	}

	return
}

func (q *Query) handleQuery() (resp string, err error) {
	trId := q.TransactionID
	if trId == "" {
		err = fmt.Errorf("Invalid Query")
		return
	}
	resp += "<status>\n"

	match_mux.Lock()
	defer match_mux.Unlock()
	conn := redis.Pool.Get()
	defer conn.Close()

	order_info, _ := redigo.Strings(conn.Do("HMGET", "order:"+trId, "amount", "origAmount"))
	log.WithFields(log.Fields{
		"order info": order_info,
	}).Info("Queried transaction")

	transactions, _ := redigo.Strings(conn.Do("LRANGE", "order-executed:"+trId, 0, -1))
	log.WithFields(log.Fields{
		"Transactions": transactions,
	}).Info("Execution history")

	if len(transactions) % 3 != 0 {
		resp = ""
		err = fmt.Errorf("Malformed Redis Data")
		return
	}

	len_trans := len(transactions) / 3
	for i := 0; i < len_trans; i++ {
		exec := ExecutedQueryResponse{Shares: transactions[3 * i], Price: transactions[3 * i + 1], Time: transactions[3 * i + 2]}
		if exec_string, err := xml.MarshalIndent(exec, "", "    "); err == nil {
			resp += string(exec_string) + "\n"
		}
	}

	remaining_amount, _ := strconv.ParseFloat(order_info[0], 64)
	if (remaining_amount > 0 ) {
		open := OpenQueryResponse{Shares: order_info[0]}
		if open_string, err := xml.MarshalIndent(open, "", "    "); err == nil {
			resp += string(open_string) + "\n"
		}
	}

	resp += "<status/>\n"

	return
}

func (order *Order) openOrder(acctId string) (transId int, err error) {
	log.Info("Creating order...")
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

func createAccount(acct *Account) error {
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
	err := redis.SetField("acct:"+acct.Id, "balance", acct.Balance)

	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Error("error setting account")
		return fmt.Errorf("Error creating account")
	}

	// TEST: - Retrieve key + field, then log
	bal, _ := redis.GetField("acct:"+acct.Id, "balance")
	bal_float, _ := strconv.ParseFloat(string(bal.([]byte)), 64)
	log.WithFields(log.Fields{
		"ID":             acct.Id,
		"Balance":        bal_float,
		"Verify_Balance": acct.Balance,
	}).Info("Created account")
	// END TEST

	return nil
}

func createSymbol(sym *Symbol) error {
	// This creates the specified symbol. The symbol tag can have one or more
	//children which are <account id="ID">NUM</account> These indicate that
	// NUM shares of the symbol being created should be placed into the account
	// with the given ID. Note that this creation is legal even if sym already
	// exists: in such a case, it is used to create more shares of that symbol
	//and add them to existing accounts.
	ex, _ := redis.Exists("sym:" + sym.Sym)
	if !ex {
		redis.Set("sym:"+sym.Sym, "")
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
			ex, _ := redis.HExists("acct:"+rcv_acct.Id+":positions", sym.Sym)

			if ex {
				amt_float, _ := strconv.ParseFloat(rcv_acct.Amount, 64)
				redis.HIncrByFloat("acct:"+rcv_acct.Id+":positions", sym.Sym, amt_float)
			} else {
				redis.SetField("acct:"+rcv_acct.Id+":positions", sym.Sym, rcv_acct.Amount)
			}

		}

		// TEST: - Retrieve key + field, then log
		bal, _ := redis.GetField("acct:"+rcv_acct.Id+":positions", sym.Sym)
		bal_float, _ := strconv.ParseFloat(string(bal.([]byte)), 64)
		log.WithFields(log.Fields{
			"ID":       rcv_acct.Id,
			"Position": bal_float,
			"Symbol":   sym.Sym,
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

							resp, _ := qry.handleQuery()
							results += resp + "\n"
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
