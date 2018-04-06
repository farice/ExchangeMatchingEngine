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
const bufferCapacity = 1

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

	// TODO(maybe) - postgres
	return
}

/// Accounts

func (m *Model) createAccount(uid string, balance string) (err error) {
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

	// postgres. Will reject if duplicate.
	sqlQuery := fmt.Sprintf("INSERT INTO account(uid, balance) VALUES('%s', %f)", uid, bal_float)
	m.submitQuery(sqlQuery)

	return
}

func (m *Model) getAccountBalance(accountID string) (balance float64, err error) {
	// Attempt fetch from redis
	log.Info("Get account balance.")
	bal, err := redis.GetField("acct:"+accountID, "balance")
	if bal != nil && err == nil {
		// TODO: Fix the error in the following line
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
			return err
		}
		// TODO: Add account to redis store
		err = redis.SetField("acct:"+accountID, "balance", newBalance)
		return
	}
	redis.HIncrByFloat("acct:"+accountID, "balance", amount)
	return
}

func (m *Model) accountExists(accountID string) (ex bool, err error) {
	ex, err = redis.Exists("acct:" + accountID)
	// TODO - Postgres

	return
}

/// Open orders

func (m *Model) createBuyOrder(uid string, accountID string, symbol string, amount float64, limit_str string, priceLimit float64) (err error) {
	// TODO: Write to cache
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
	conn := redis.Pool.Get()
	defer conn.Close()
	_, err = conn.Do("HMSET", "order-cancel:"+trId, "amount", amt_f, "time", time)

	// TODO - postgres

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
	// TODO: Get from redis
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
	// TODO - Fix query (syntax error)
	sqlQuery := fmt.Sprintf(`DELETE FROM buy_order WHERE uid='%s'`, uid)
	sql_err := m.db.QueryRow(sqlQuery).Scan()
	if sql_err != nil {
		log.Error(fmt.Sprintf(`SQL database error: %v -- query: %s`, err, sqlQuery))
	}
	return
}

func (m *Model) createSellOrder(uid string, accountID string, symbol string, amount float64, limit_str string, priceLimit float64) (err error) {
	// TODO: Write to redis
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
	sql_err := m.db.QueryRow(sqlQuery).Scan()
	if sql_err != nil {
		log.Error(fmt.Sprintf(`SQL database error: %v -- query: %s`, err, sqlQuery))
	}
	return
}

func (m *Model) getMaximumBuyOrder(symbol string, priceLimit float64) (uid []string, err error) {
	// TODO: Get from redis
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
	// TODO: Find in cache
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

/// Transactions

func (m *Model) transactionExists(transId string) (ex bool, err error) {
	ex, err = redis.Exists("order:" + transId)

	// TODO - postgres

	return
}
func (m *Model) createTransaction(transId string, acctId string, sym string, limit string, amount string, transactionTime time.Time) (err error) {
	// TODO: Create in redis
	conn := redis.Pool.Get()
	defer conn.Close()
	_, err = conn.Do("HMSET", "order:"+transId, "account", acctId, "symbol", sym, "limit", limit, "amount", amount, "origAmount", amount)

	amountFloat, _ := strconv.ParseFloat(amount, 64)
	limitFloat, _ := strconv.ParseFloat(limit, 64)

	formattedTime := transactionTime.String()

	sqlQuery := fmt.Sprintf(`INSERT INTO transaction(symbol, amount, price, transaction_time) VALUES('%s', %f, %f, '%s')`, sym, amountFloat, limitFloat, formattedTime)
	m.submitQuery(sqlQuery)

	return
}

func (m *Model) getTransaction(trId string) (data []string, err error) {
	conn := redis.Pool.Get()
	defer conn.Close()
	data, err = redigo.Strings(conn.Do("HMGET", "order:"+trId, "account", "symbol", "limit", "amount", "origAmount"))
	// TODO - postgres

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
	} else {
		err = redis.SetField("acct:"+accountID+":positions", symbol, amount)
	}

	// TODO - Postgres

	return
}

func (m *Model) addSharesToPosition(accountID string, symbol string, amount float64) (err error) {

	_, err = redis.HIncrByFloat("acct:"+accountID+":positions", symbol, amount)

	sqlQuery := fmt.Sprintf(`UPDATE position SET amount=amount+%f WHERE account_id = '%s' AND symbol='%s'`, amount, accountID, symbol)
	m.submitQuery(sqlQuery)

	return
}

func (m *Model) updatePosition(accountID string, symbol string, amount float64) (err error) {
	positionExists := false
	fetchQuery := fmt.Sprintf(`SELECT amount FROM position WHERE account_id='%s' AND symbol='%s'`, accountID, symbol)
	var currentAmount float64
	err = m.db.QueryRow(fetchQuery).Scan(&currentAmount)
	positionExists = err == nil
	if !positionExists {
		// TODO: Write to cache
		sqlQuery := fmt.Sprintf(`INSERT INTO position(account_id, symbol, amount) VALUES('%s', '%s', %f)`, accountID, symbol, amount)
		m.submitQuery(sqlQuery)
		return nil
	}
	sqlQuery := fmt.Sprintf(`UPDATE position SET amount=%f WHERE account_id = '%s' AND symbol='%s'`, currentAmount+amount, accountID, symbol)
	m.submitQuery(sqlQuery)
	return nil
}

func (m *Model) removePosition(accountID string, symbol string) (err error) {
	// TODO: Update cache
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
		println("QUERY: ", sqlQuery)
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
