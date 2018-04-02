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
	commands []string
}

/// Accounts

func (m *Model) createAccount(balance float64) (uid string, err error) {
	uid = ksuid.New().String()
	err = m.createAccountWithID(uid, balance)
	return uid, err
}

func (m *Model) createAccountWithID(uid string, balance float64) (err error) {
	sqlQuery := fmt.Sprintf("INSERT INTO account(uid, balance) VALUES(%s, %f)", uid, balance)
	m.submitQuery(sqlQuery)
	return nil
}

func (m *Model) getAccountBalance(accountID string) (balance float64, err error) {

	return 0, nil
}

/// Orders

func (m *Model) submitBuyOrder(accountID string, amount float64, limit float64) (orderID string, err error) {

	return "", nil
}

func (m *Model) submitSellOrder(accountID string, amount float64, limit float64) (orderID string, err error) {

	return "", nil
}

/// Symbols

func (m *Model) createOrUpdateSymbol(symbol string, shares float64) (err error) {
	// ex, _ := redis.Exists("sym:" + sym.Sym)
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
		sqlQuery := fmt.Sprintf("UPDATE symbol SET shares=%f WHERE name=%s", shares, symbol)
		m.submitQuery(sqlQuery)
	}
	return nil
}

func (m *Model) getSymbolSharesTotal(symbol string) (symbolExists bool, share float64, err error) {
	return false, 0, nil
}

/// Positions

func (m *Model) createPosition(accountID string, symbol string, amount float64) {

}

func (m *Model) removePosition(accountID string, symbol string, amount float64) {

}

func (m *Model) submitQuery(query string) {
	print(query)
	_, err := m.db.Exec(query)
	if err != nil {
		log.Error("SQL database error: ", err)
	}
}
