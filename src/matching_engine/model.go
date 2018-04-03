package main

import (
	"database/sql"
	"fmt"

	// "github.com/farice/EME/redis"
	_ "github.com/lib/pq"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
)

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
	// TODO: Attempt fetch from redis

	// If must check postgres
	sqlQuery := fmt.Sprint(`SELECT balance FROM account WHERE uid='%s'`, accountID)
	err = m.db.QueryRow(sqlQuery).Scan(&balance)
	if err != nil {
		// log.Error("Error in fetch account balance: ", err)
		return -1, err
	}
	return balance, nil
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
	sqlQuery = fmt.Sprintf(`DELETE * from buy_order WHERE uid='%s'`, uid)
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

func (m *Model) cancelSellOrder(uid string, accountID string) {
	// TODO: Find in cache

	// If must go to db
	sqlQuery = fmt.Sprintf(`DELETE * from sell_order WHERE uid='%s'`, uid)
	err = m.db.QueryRow(sqlQuery).Scan()
	return err
}

func (m *Model) getMinimumSellOrder(symbol string, maximum float64) (uid string, err error) {
	// TODO: Find in cache

	// If must go to db
	sqlQuery = fmt.Sprintf(`SELECT TOP 1 uid FROM sell_order WHERE limit=(SELECT MIN(limit) FROM sell_order WHERE symbol=%s)  AND limit <= %f`, symbol, limit)
	err = m.db.Query(sqlQuery).Scan(&uid)
	if err != nil {
		return nil, err
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

func (m *Model) createPosition(accountID string, symbol string, amount float64) {
	// TODO: Write to cache
	uid := ksuid.New().String()
	sqlQuery := fmt.Sprintf(`INSERT INTO position(uid, account_id, symbol, amount) VALUES('$s', '%s', '%s', %f)`, uid, accountID, symbol, amount)
	m.submitQuery(sqlQuery)
}

func (m *Model) removePosition(uid string) {
	// TODO: Update cache
	sqlQuery := fmt.Sprintf(`DELETE FROM position WHERE uid='%s'`, uid)
	m.submitQuery(sqlQuery)
}

/// Implementation / private

func (m *Model) submitQuery(query string) {
	m.commands <- query
}

func (m *Model) executeQueries() {
	log.Info(fmt.Sprintf("Flushing SQL commands. There are %d commands in the buffer.", len(m.commands)))
	for len(m.commands) > 0 {
		_, err := m.db.Exec(<-m.commands)
		if err != nil {
			log.Error("SQL database error: ", err)
		}
	}
}
