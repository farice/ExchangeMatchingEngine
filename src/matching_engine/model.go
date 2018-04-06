package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/farice/EME/redis"
	redigo "github.com/gomodule/redigo/redis"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

const user = "postgres"
const dbname = "exchange"
const sslmode = "disable"
const bufferCapacity = 10

// Singleton approach found here: http://marcio.io/2015/07/singleton-pattern-in-go/#comment-2132217074
var initialized uint32
var instance *Model
var mu sync.Mutex

func SharedModel() *Model {

	if atomic.LoadUint32(&initialized) == 1 {
		return instance
	}

	mu.Lock()
	defer mu.Unlock()

	if initialized == 0 {
		db, err := sql.Open("postgres", dbInfoString())
		if err != nil {
			log.Fatal("DATABASE ERROR: ", err)
			return nil
		}
		instance = &Model{db, make(chan string, bufferCapacity)}
		atomic.StoreUint32(&initialized, 1)
	}
	return instance
}

func dbInfoString() (info string) {
	return fmt.Sprintf("user=%s dbname=%s sslmode=%s host=db", user, dbname, sslmode)
}

// var shared Model = {db, make(chan string, 100)}

// Model provides access to the application's data layer.
type Model struct {
	db       *sql.DB
	commands chan string
}

// Counter

func (m *Model) incTransactionCounter() (ct int, err error) {
	ct, err = redis.Incr("TransactionCounter")
	return
}

/// Accounts

func (m *Model) createAccount(uid string, balance string) (err error) {
	LogMethodTimeElapsed("model.createAccount", time.Now())
	// This creates a new account with the given unique ID and balance (in USD).
	// The account has no positions. Attempting to create an account that already
	// exists is an error.
	ex, _ := redis.Exists("acct:" + uid)
	if uid == "" {
		// TODO: - Throw and handle error
		return nil
	}
	if ex {
		log.WithFields(log.Fields{
			"ID": uid,
		}).Info("Duplicate account")
		return fmt.Errorf("Duplicate account")
	}

	// Redis HMSET, maps key to hashmap of fields to values
	err = redis.SetField("acct:"+uid, "balance", balance)

	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Error("error setting account")
		err = fmt.Errorf("Error creating account")
		return
	}

	// TEST: - Retrieve key + field, then log
	bal, _ := redis.GetField("acct:"+uid, "balance")
	bal_float, _ := strconv.ParseFloat(string(bal.([]byte)), 64)
	log.WithFields(log.Fields{
		"ID":             uid,
		"Balance":        bal_float,
		"Verify_Balance": balance,
	}).Info("Created account")
	// END TEST

	balanceFloat, convErr := strconv.ParseFloat(balance, 64)
	if convErr != nil {
		log.Error("Failed conversion of string to float: ", convErr)
		log.Error("String failed to convert: ", balance)
	} else {
		log.Infof("Converted string %s to float %f", balance, balanceFloat)
	}
	// postgres. Will reject if duplicate.
	sqlQuery := fmt.Sprintf(`INSERT INTO account(uid, balance) VALUES('%s', %f)`, uid, bal_float)
	m.submitQuery(sqlQuery)
	return
}

func (m *Model) getAccountBalance(accountID string) (balance float64, err error) {
	LogMethodTimeElapsed("model.getAccountBalance", time.Now())
	// Attempt fetch from redis
	log.Info("Get account balance.")
	bal, err := redis.GetField("acct:"+accountID, "balance")
	if bal != nil && err == nil {
		balance, err = strconv.ParseFloat(string(bal.([]byte)), 64)
		return balance, nil
	}

	// If must check postgres
	sqlQuery := fmt.Sprintf(`SELECT balance FROM account WHERE uid='%s'`, accountID)
	err = m.db.QueryRow(sqlQuery).Scan(&balance)
	if err != nil {
		log.Error(fmt.Sprintf(`SQL database error: %v -- query: %s`, err, sqlQuery))
		return -1, err
	}
	return balance, nil
}

func (m *Model) addAccountBalance(accountID string, amount float64) (err error) {
	ex, _ := redis.HExists("acct:"+accountID, "balance")
	if ex == false {
		// If not in cache
		var newBalance float64
		sqlQuery := fmt.Sprintf(`UPDATE account SET balance=balance+%f WHERE uid='%s' RETURNING balance`, amount, accountID)
		err = m.db.QueryRow(sqlQuery).Scan(&newBalance)
		if err != nil {
			// Likely non-existent account
			log.Error(fmt.Sprintf(`SQL database error: %v -- query: %s`, err, sqlQuery))
			err = fmt.Errorf("Account does not exist")
			return
		}
		err = redis.SetField("acct:"+accountID, "balance", newBalance)
		return
	}
	redis.HIncrByFloat("acct:"+accountID, "balance", amount)

	var newBalance float64
	sqlQuery := fmt.Sprintf(`UPDATE account SET balance=balance+%f WHERE uid='%s' RETURNING balance`, amount, accountID)
	sqlErr := m.db.QueryRow(sqlQuery).Scan(&newBalance)
	if sqlErr != nil {
		log.Error(fmt.Sprintf(`SQL database error: %v -- query: %s`, err, sqlQuery))
	}

	return
}

func (m *Model) accountExists(accountID string) (ex bool, err error) {
	log.Info("Account Exists")
	ex, err = redis.Exists("acct:" + accountID)
	if !ex {
		sqlQuery := fmt.Sprintf(`SELECT * FROM account WHERE uid='%s'`, accountID)
		var balance float64
		sqlErr := m.db.QueryRow(sqlQuery).Scan(&balance)
		if sqlErr == nil {
			// Exists in DB
			bal_string := strconv.FormatFloat(balance, 'E', -1, 64)
			err = redis.SetField("acct:"+accountID, "balance", bal_string)
			return true, nil
		}
	}

	return
}

/// Open orders

func (m *Model) createBuyOrder(uid string, accountID string, symbol string, amount float64, limit_str string, priceLimit float64) (err error) {

	log.Info("Create Buy Order")

	err = redis.Zadd("open-buy:"+symbol, limit_str, uid)

	sqlQuery := fmt.Sprintf(`INSERT INTO buy_order(uid, account_id, symbol, amount, price_limit) VALUES('%s', '%s', '%s', %f, %f);`, uid, accountID, symbol, amount, priceLimit)
	m.submitQuery(sqlQuery)
	return err
}

func (m *Model) updateBuyOrderAmount(uid string, newAmount float64) (err error) {

	err = redis.SetField("order:"+uid, "amount", newAmount)

	sqlQuery := fmt.Sprintf(`UPDATE buy_order SET amount=%f WHERE uid = '%s'`, newAmount, uid)
	m.submitQuery(sqlQuery)
	return
}

// fills cancellation details
func (m *Model) cancelOrder(trId string, amt_f float64, time string) (err error) {
	log.Info("Cancel Order")
	conn := redis.Pool.Get()
	defer conn.Close()
	_, err = conn.Do("HMSET", "order-cancel:"+trId, "amount", amt_f, "time", time)

	// Postgres removes the open order
	var sqlQuery string
	if amt_f > 0 {
		sqlQuery = fmt.Sprintf(`DELETE FROM buy_order WHERE uid='%s'`, trId)
	} else {
		sqlQuery = fmt.Sprintf(`DELETE FROM sell_order WHERE uid='%s'`, trId)
	}
	m.submitQuery(sqlQuery)

	return
}

// is order cancelled
func (m *Model) orderCancelled(trId string) (ex bool, err error) {
	ex, err = redis.Exists("order-cancel:" + trId)
	// TODO - postgres

	return
}

func (m *Model) getCancelledOrderDetails(trId string) (cancel_info []string, err error) {
	conn := redis.Pool.Get()
	defer conn.Close()

	cancel_info, err = redigo.Strings(conn.Do("HMGET", "order-cancel:"+trId, "amount", "time"))

	// TODO - postgres

	return
}

// Update Executed shares list
func (m *Model) executedOrder(trId string, amount float64, limit float64, time string) (err error) {
	conn := redis.Pool.Get()
	defer conn.Close()

	_, err = conn.Do("RPUSH", "order-executed:"+trId, amount, limit, time)
	// TODO - postgres

	return
}

// from list constructed directly above
func getPartialExecutions(trId string) (transactions []string, err error) {
	conn := redis.Pool.Get()
	defer conn.Close()

	transactions, err = redigo.Strings(conn.Do("LRANGE", "order-executed:"+trId, 0, -1))

	// TODO = postgres

	return
}

func (m *Model) closeOpenBuyOrder(uid string, sym string) (err error) {
	log.Info("Close open buy order")
	conn := redis.Pool.Get()
	defer conn.Close()
	// num deleted
	var num int
	num, err = redigo.Int(conn.Do("ZREM", "open-buy:"+sym, uid))

	log.WithFields(log.Fields{
		"transId": uid,
		"error":   err,
		"deleted": num,
	}).Info("Removed open order from sorted set")

	// If have to go to db
	sqlQuery := fmt.Sprintf(`DELETE FROM buy_order WHERE uid='%s'`, uid)
	m.submitQuery(sqlQuery)
	// sqlErr := m.db.QueryRow(sqlQuery).Scan()
	// if sqlErr != nil {
	// 	log.Error(fmt.Sprintf(`SQL database error: %v -- query: %s`, err, sqlQuery))
	// }
	return
}

func (m *Model) createSellOrder(uid string, accountID string, symbol string, amount float64, limit_str string, priceLimit float64) (err error) {
	err = redis.Zadd("open-sell:"+symbol, limit_str, uid)

	sqlQuery := fmt.Sprintf(`INSERT INTO sell_order(uid, account_id, symbol, amount, price_limit) VALUES('%s', '%s', '%s', %f, %f);`, uid, accountID, symbol, amount, priceLimit)
	m.submitQuery(sqlQuery)
	return err
}

func (m *Model) updateSellOrderAmount(uid string, newAmount float64) (err error) {
	err = redis.SetField("order:"+uid, "amount", newAmount)

	sqlQuery := fmt.Sprintf(`UPDATE sell_order SET amount=%f WHERE uid = '%s'`, newAmount, uid)
	m.submitQuery(sqlQuery)
	return
}

func (m *Model) closeOpenSellOrder(uid string, sym string) (err error) {
	log.Info("Close Open sell order")
	conn := redis.Pool.Get()
	defer conn.Close()
	// num deleted
	var num int
	num, err = redigo.Int(conn.Do("ZREM", "open-sell:"+sym, uid))

	log.WithFields(log.Fields{
		"transId": uid,
		"error":   err,
		"deleted": num,
	}).Info("Removed open order from sorted set")

	// If must go to db
	sqlQuery := fmt.Sprintf(`DELETE FROM sell_order WHERE uid='%s'`, uid)
	m.submitQuery(sqlQuery)
	// sqlErr := m.db.QueryRow(sqlQuery).Scan()
	// if sqlErr != nil {
	// 	log.Error(fmt.Sprintf(`SQL database error: %v -- query: %s`, err, sqlQuery))
	// }
	return
}

func (m *Model) getMaximumBuyOrder(symbol string, priceLimit float64) (uid []string, err error) {
	uid, err = redis.Zrange("open-buy:"+symbol, -1, -1, true)
	if len(uid) > 0 && uid[0] != "" {
		return
	}

	// TODO - Fix syntax error
	/*
		sqlQuery := fmt.Sprintf(`SELECT TOP 1 uid FROM buy_order WHERE price_limit=(SELECT MAX(price_limit) FROM buy_order WHERE symbol=%s)  AND price_limit >= %f`, symbol, priceLimit)
		err = m.db.QueryRow(sqlQuery).Scan(&uid)
		if err != nil {
			log.Error("SQL Error: %s -- Query: %s", err, sqlQuery)
			return
		}
	*/
	return

}

func (m *Model) getMinimumSellOrder(symbol string, priceLimit float64) (uid []string, err error) {
	uid, err = redis.Zrange("open-sell:"+symbol, 0, 0, true)
	if len(uid) > 0 && uid[0] != "" {
		return
	}

	// TODO - Fix syntax error
	/*
		// If must go to db
		sqlQuery := fmt.Sprintf(`SELECT TOP 1 uid FROM sell_order WHERE price_limit=(SELECT MIN(price_limit) FROM sell_order WHERE symbol=%s)  AND price_limit <= %f`, symbol, priceLimit)
		err = m.db.QueryRow(sqlQuery).Scan(&uid)
		if err != nil {
			log.Error(fmt.Sprintf(`SQL database error: %v -- query: %s`, err, sqlQuery))
			return
		} */
	return
}

/// Orders

func (m *Model) transactionExists(transID string) (ex bool, err error) {
	ex, err = redis.Exists("order:" + transID)

	if !ex {
		sqlQuery := fmt.Sprintf(`SELECT * FROM sell_order WHERE uid='%s'`, transID)
		sqlErr := m.db.QueryRow(sqlQuery).Scan()
		if sqlErr == nil {
			return true, nil
		}
		sqlQuery = fmt.Sprintf(`SELECT * FROM buy_order WHERE uid='%s'`, transID)
		sqlErr = m.db.QueryRow(sqlQuery).Scan()
		if sqlErr == nil {
			return true, nil
		}
		sqlQuery = fmt.Sprintf(`SELECT * FROM transaction WHERE uid='%s'`, transID)
		sqlErr = m.db.QueryRow(sqlQuery).Scan()
		if sqlErr == nil {
			return true, nil
		}
	}

	return
}

func (m *Model) createOrder(transID string, acctID string, sym string, limit string, amount string, transactionTime time.Time) (err error) {
	log.Info("mode.createOrder")
	LogMethodTimeElapsed("mode.createOrder", time.Now())
	conn := redis.Pool.Get()
	defer conn.Close()
	_, err = conn.Do("HMSET", "order:"+transID, "account", acctID, "symbol", sym, "limit", limit, "amount", amount, "origAmount", amount)

	// No need to create anything in postgres here.

	// amountFloat, _ := strconv.ParseFloat(amount, 64)
	// limitFloat, _ := strconv.ParseFloat(limit, 64)

	// formattedTime := transactionTime.String()

	// sqlQuery := fmt.Sprintf(`INSERT INTO transaction(uid, symbol, amount, price, transaction_time) VALUES('%s', '%s', %f, %f, '%s')`, transID, sym, amountFloat, limitFloat, formattedTime)
	// m.submitQuery(sqlQuery)

	return
}

// Get order or closed transaction.
func (m *Model) getOrder(orderID string) (data []string, err error) {
	conn := redis.Pool.Get()
	defer conn.Close()
	data, err = redigo.Strings(conn.Do("HMGET", "order:"+orderID, "account", "symbol", "limit", "amount", "origAmount"))

	// postgres
	var uid string
	var accountID string
	var symbol string
	limit := 0.0
	var amount float64
	// originalAmount := 0.0
	var time string

	sqlQuery := fmt.Sprintf(`SELECT * FROM sell_order WHERE uid='%s'`, orderID)
	sqlErr := m.db.QueryRow(sqlQuery).Scan(&uid, &accountID, &symbol, &limit, &amount)
	if sqlErr == nil {
		limitString := strconv.FormatFloat(limit, 'E', -1, 64)
		amountString := strconv.FormatFloat(amount, 'E', -1, 64)
		data = []string{
			uid,
			accountID,
			symbol,
			limitString,
			amountString,
		}
		return data, err
	}
	sqlQuery = fmt.Sprintf(`SELECT * FROM buy_order WHERE uid='%s'`, orderID)
	sqlErr = m.db.QueryRow(sqlQuery).Scan(&uid, &accountID, &symbol, &limit, &amount)
	if sqlErr == nil {
		limitString := strconv.FormatFloat(limit, 'E', -1, 64)
		amountString := strconv.FormatFloat(amount, 'E', -1, 64)
		data = []string{
			uid,
			accountID,
			symbol,
			limitString,
			amountString,
		}
		return data, err
	}
	sqlQuery = fmt.Sprintf(`SELECT * FROM transaction WHERE uid='%s'`, orderID)
	sqlErr = m.db.QueryRow(sqlQuery).Scan(&uid, &symbol, &amount, &limit, &time)
	if sqlErr == nil {
		limitString := strconv.FormatFloat(limit, 'E', -1, 64)
		amountString := strconv.FormatFloat(amount, 'E', -1, 64)
		data = []string{
			uid,
			accountID,
			symbol,
			limitString,
			amountString,
		}
		return data, err
	}

	return
}

/// Symbols

func (m *Model) createOrUpdateSymbol(symbol string) (err error) {
	ex, _ := redis.Exists("sym:" + symbol)
	if !ex {
		redis.Set("sym:"+symbol, "")
		sqlQuery := fmt.Sprintf(`INSERT INTO symbol(name) VALUES('%s');`, symbol)
		m.submitQuery(sqlQuery)
	}
	return
}

/// Positions

// Add shares to existing position or set shares to value if dne
func (m *Model) addOrSetSharesToPosition(accountID string, symbol string, amount float64) (err error) {
	ex, _ := redis.HExists("acct:"+accountID+":positions", symbol)

	if ex {
		return m.addSharesToPosition(accountID, symbol, amount)
	}
	// Check if exists in postgres
	fetchQuery := fmt.Sprintf(`SELECT amount FROM position WHERE account_id='%s' AND symbol='%s'`, accountID, symbol)
	var currentAmount float64
	currentAmount = amount
	sqlErr := m.db.QueryRow(fetchQuery).Scan(&currentAmount)
	err = redis.SetField("acct:"+accountID+":positions", symbol, currentAmount)
	// If was in the db, still need to add the amount
	if sqlErr == nil {
		m.addSharesToPosition(accountID, symbol, amount)
	} else {
		// Create in DB
		sqlQuery := fmt.Sprintf(`INSERT INTO position(account_id, symbol, amount) VALUES('%s', '%s', %f)`, accountID, symbol, amount)
		m.submitQuery(sqlQuery)
	}

	return
}

func (m *Model) addSharesToPosition(accountID string, symbol string, amount float64) (err error) {
	_, err = redis.HIncrByFloat("acct:"+accountID+":positions", symbol, amount)

	sqlQuery := fmt.Sprintf(`UPDATE position SET amount=amount+%f WHERE account_id = '%s' AND symbol='%s'`, amount, accountID, symbol)
	m.submitQuery(sqlQuery)

	return
}

// func (m *Model) updatePosition(accountID string, symbol string, amount float64) (err error) {
// 	positionExists := false
// 	fetchQuery := fmt.Sprintf(`SELECT amount FROM position WHERE account_id='%s' AND symbol='%s'`, accountID, symbol)
// 	var currentAmount float64
// 	err = m.db.QueryRow(fetchQuery).Scan(&currentAmount)
// 	positionExists = err == nil
// 	if !positionExists {
// 		// TODO: Write to cache
// 		sqlQuery := fmt.Sprintf(`INSERT INTO position(account_id, symbol, amount) VALUES('%s', '%s', %f)`, accountID, symbol, amount)
// 		m.submitQuery(sqlQuery)
// 		return nil
// 	}
// 	sqlQuery := fmt.Sprintf(`UPDATE position SET amount=%f WHERE account_id = '%s' AND symbol='%s'`, currentAmount+amount, accountID, symbol)
// 	m.submitQuery(sqlQuery)
// 	return nil
// }

func (m *Model) removePosition(accountID string, symbol string) (err error) {
	sqlQuery := fmt.Sprintf(`DELETE FROM position WHERE account_id='%s' AND symbol='%s';`, accountID, symbol)
	m.submitQuery(sqlQuery)
	return nil
}

func (m *Model) getPositionAmount(accountID string, symbol string) (amount float64, err error) {
	var bal interface{}
	bal, err = redis.GetField("acct:"+accountID+":positions", symbol)
	if err != nil {
		return
	}
	if bal != nil {
		amount, err = strconv.ParseFloat(string(bal.([]byte)), 64)
		return
	} else {
		sqlQuery := fmt.Sprintf(`SELECT amount FROM position WHERE account_id='%s' AND symbol='%s';`, accountID, symbol)
		// println("QUERY: ", sqlQuery)
		err = m.db.QueryRow(sqlQuery).Scan(&amount)

		if err != nil {
			err = fmt.Errorf("User owns no shares of %s", symbol)
			log.Error(fmt.Sprintf(`SQL database error: %v -- query: %s`, err, sqlQuery))
			return
		}
	}

	return

}

/// Implementation / private

func confirmDelete(deleteQuery string) {
	print("Deleting entity with query: ")
	println(deleteQuery)
}

func (m *Model) submitQuery(query string) {
	defer LogMethodTimeElapsed("model.submitQuery", time.Now())
	m.commands <- query
	if len(m.commands) >= bufferCapacity {
		m.executeQueries()
	}
}

func (m *Model) executeQueries() {
	defer LogMethodTimeElapsed("model.executeQueries", time.Now())
	log.Info(fmt.Sprintf("Flushing SQL commands. There are %d commands in the buffer.", len(m.commands)))
	for len(m.commands) > 0 {
		s := <-m.commands
		println("EXECUTING QUERY: ", s)
		var query string
		isDelete := strings.HasPrefix(s, "DELETE")
		if isDelete {
			query = s
		}
		// reportProblem := func(ev pq.ListenerEventType, err error) {
		// 	if err != nil {
		// 		fmt.Println(err.Error())
		// 	}
		// }

		// TODO: Set up listener for record update to cache
		// listener := pq.NewListener(dbInfoString(), 10*time.Second, time.Minute, reportProblem)
		// listener := pq.NewListener()
		_, err := m.db.Exec(s)
		if err != nil {
			log.Error(fmt.Sprintf(`SQL database error: %v -- query: %s`, err, s))
		}
		if isDelete {
			// Dispatch to other thread?
			go confirmDelete(query)
		}
	}
}
