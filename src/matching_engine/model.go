package main

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	//"time"

	"github.com/farice/EME/redis"
	_ "github.com/lib/pq"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
)

const user = "andrewbihl"
const dbname = "exchange"
const sslmode = "disable"

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
		instance = &Model{db, make(chan string, 100)}
		atomic.StoreUint32(&initialized, 1)
	}
	return instance
}

func dbInfoString() (info string) {
	return fmt.Sprintf("user=%s dbname=%s sslmode=%s", user, dbname, sslmode)
}

// var shared Model = {db, make(chan string, 100)}

// Model provides access to the application's data layer.
type Model struct {
	db       *sql.DB
	commands chan string
}

/// Accounts

func (m *Model) createAccount(balance float64) (uid string, err error) {
	// TODO: Write to cache
	uid = ksuid.New().String()
	err = m.createAccountWithID(uid, balance)
	return uid, err
}

func (m *Model) createAccountWithID(uid string, balance float64) (err error) {
	// TODO: Write to cache
	sqlQuery := fmt.Sprintf("INSERT INTO account(uid, balance) VALUES('%s', %f)", uid, balance)
	m.submitQuery(sqlQuery)
	return nil
}

func (m *Model) getAccountBalance(accountID string) (balance float64, err error) {
	// Attempt fetch from redis
	bal, _ := redis.GetField("acct:"+accountID, "balance")
	if bal != nil {
		// TODO: Fix the error in the following line
		// balance = strconv.ParseFloat(string(bal.([]byte)), 64)
		return balance, nil
	}

	// If must check postgres
	sqlQuery := fmt.Sprint(`SELECT balance FROM account WHERE uid='%s'`, accountID)
	err = m.db.QueryRow(sqlQuery).Scan(&balance)
	if err != nil {
		// log.Error("Error in fetch account balance: ", err)
		return -1, err
	}
	return balance, nil
}

func (m *Model) addAccountBalance(accountID string, amount float64) (err error) {
	ex, _ := redis.HExists("acct:"+accountID, "balance")
	if ex == false {
		// Should return err if cannot find row.
		var currentAmount float64
		err = m.db.QueryRow(fmt.Sprintf(`GET balance FROM account WHERE uid='%s'`, accountID)).Scan(&currentAmount)
		// If user does not exist
		if err != nil {
			err = fmt.Errorf("User %s does not exist", accountID)
			return err
		}
		err = m.db.QueryRow(fmt.Sprintf(`UPDATE symbol SET balance=%f WHERE uid='%s'`, currentAmount+amount, accountID)).Scan()
		// TODO: Add account to redis store

		return nil
	}
	redis.HIncrByFloat("acct:"+accountID, "balance", amount)
	return nil
}

/// Orders

func (m *Model) submitBuyOrder(accountID string, symbol string, amount float64, limit float64) (orderID string, err error) {
	// TODO: Write to cache
	uid := ksuid.New().String()
	sqlQuery := fmt.Sprintf(`INSERT INTO buy_order(uid, account_id, symbol, amount, limit) VALUES('%s', '%s', '%s', %f, %f);`, uid, accountID, symbol, amount, limit)
	m.submitQuery(sqlQuery)
	return uid, err
}

func (m *Model) cancelBuyOrder(uid string, accountID string) (err error) {
	// TODO: Get from cache

	// If have to go to db
	sqlQuery := fmt.Sprintf(`DELETE * from buy_order WHERE uid='%s'`, uid)
	err = m.db.QueryRow(sqlQuery).Scan()
	return err
}

func (m *Model) submitSellOrder(accountID string, symbol string, amount float64, limit float64) (orderID string, err error) {
	// TODO: Write to cache
	uid := ksuid.New().String()
	sqlQuery := fmt.Sprintf(`INSERT INTO buy_order(uid, account_id, symbol, amount, limit) VALUES('%s', '%s', '%s', %f, %f);`, uid, accountID, symbol, amount, limit)
	m.submitQuery(sqlQuery)
	return uid, err
}

func (m *Model) cancelSellOrder(uid string, accountID string) (err error) {
	// TODO: Find in cache

	// If must go to db
	sqlQuery := fmt.Sprintf(`DELETE * from sell_order WHERE uid='%s'`, uid)
	err = m.db.QueryRow(sqlQuery).Scan()
	return
}

func (m *Model) getMinimumSellOrder(symbol string, limit float64, maximum float64) (uid string, err error) {
	// TODO: Find in cache

	// If must go to db
	sqlQuery := fmt.Sprintf(`SELECT TOP 1 uid FROM sell_order WHERE limit=(SELECT MIN(limit) FROM sell_order WHERE symbol=%s)  AND limit <= %f`, symbol, limit)
	err = m.db.QueryRow(sqlQuery).Scan(&uid)
	if err != nil {
		return "", err
	}
	return uid, nil
}

/// Symbols

func (m *Model) createOrUpdateSymbol(symbol string, shares float64) (err error) {
	exists, totalShares, err := m.getSymbolSharesTotal(symbol)
	if !exists {
		// TODO: Set amount in redis
		// redis.Set("sym:"+sym.Sym, "")
		if err != nil {
			return err
		}
		sqlQuery := fmt.Sprintf(`INSERT INTO symbol(name, shares) VALUES('%s', %f);`, symbol, totalShares+shares)
		m.submitQuery(sqlQuery)
	} else {
		// Update amount
		sqlQuery := fmt.Sprintf("UPDATE symbol SET shares=%f WHERE name='%s'", shares, symbol)
		m.submitQuery(sqlQuery)
	}
	return nil
}

func (m *Model) getSymbolSharesTotal(symbol string) (symbolExists bool, shares float64, err error) {
	// TODO: Check redis
	// ex, _ := redis.Exists("sym:" + sym.Sym)

	// If need to ask postgres
	sqlQuery := fmt.Sprintf(`SELECT shares FROM symbol WHERE name='%s';`, symbol)
	err = m.db.QueryRow(sqlQuery).Scan(&shares)
	if err != nil {
		return false, 0, err
	}
	return true, shares, nil
}

/// Positions

func (m *Model) updatePosition(accountID string, symbol string, amount float64) (uid string) {
	positionExists := false
	if !positionExists {
		// TODO: Write to cache
		uid = ksuid.New().String()
		sqlQuery := fmt.Sprintf(`INSERT INTO position(uid, account_id, symbol, amount) VALUES('%s', '%s', '%s', %f)`, uid, accountID, symbol, amount)
		m.submitQuery(sqlQuery)
		return uid
	}
	return ""
}

func (m *Model) removePosition(uid string) {
	// TODO: Update cache
	sqlQuery := fmt.Sprintf(`DELETE FROM position WHERE uid='%s'`, uid)
	m.submitQuery(sqlQuery)
}

/// Implementation / private

func confirmDelete(deleteQuery string) {
	print("Deleting entity with query: ")
	println(deleteQuery)
}

func (m *Model) submitQuery(query string) {
	m.commands <- query
}

func (m *Model) executeQueries() {
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
			log.Error("SQL database error: ", err)
		}
		if isDelete {
			// Dispatch to other thread?
			go confirmDelete(query)
		}
	}
}
