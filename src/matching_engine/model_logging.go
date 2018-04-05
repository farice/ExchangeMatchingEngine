package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func outputDatabaseStateTruncated(rowLimit int) {
	SharedModel().executeQueries()

	outputAccounts(rowLimit)
	outputSymbols(rowLimit)
	outputPositions(rowLimit)
	outputBuyOrders(rowLimit)
	outputSellOrders(rowLimit)
}

func outputAccounts(rowLimit int) {
	tableName := "account"
	rows, err := SharedModel().db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT %d", tableName, rowLimit))
	if err != nil {
		log.Info("Error attempting to print accounts: ", err)
		return
	}
	var uid string
	var balance float64
	println("\n#######  ACCOUNTS (max 50):")
	for rows.Next() {
		err = rows.Scan(&uid, &balance)
		println(fmt.Sprintf("AccountID: %s -- Balance: %f", uid, balance))
	}
}

func outputSymbols(rowLimit int) {
	tableName := "symbol"
	var symbol string
	var shares float64

	rows, err := SharedModel().db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT %d", tableName, rowLimit))
	if err != nil {
		log.Info("Error attempting to print symbols: ", err)
		return
	}
	println("\n#######  SYMBOLS (max 50): ")
	for rows.Next() {
		err = rows.Scan(&symbol, &shares)
		println(fmt.Sprintf("Symbol: %s -- Shares: %f", symbol, shares))
	}
}

func outputPositions(rowLimit int) {
	tableName := "position"
	var accountID string
	var symbol string
	var amount float64

	rows, err := SharedModel().db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT %d", tableName, rowLimit))
	if err != nil {
		log.Info("Error attempting to print positions: ", err)
		return
	}
	println("\n#######  POSITIONS (max 50): ")
	for rows.Next() {
		err = rows.Scan(&accountID, &symbol, &amount)
		println(fmt.Sprintf("AccountID: %s -- Symbol: %s -- Amount: %f", accountID, symbol, amount))
	}
}

func outputBuyOrders(rowLimit int) {
	tableName := "buy_order"
	var uid string
	var accountID string
	var symbol string
	var priceLimit float64
	var amount float64

	sqlQuery := fmt.Sprintf("SELECT * FROM %s LIMIT %d", tableName, rowLimit)
	print("QUERY: ", sqlQuery)
	rows, err := SharedModel().db.Query(sqlQuery)
	if err != nil {
		log.Info("Error attempting to print buy orders: ", err)
		return
	}
	println("\n#######  BUY ORDERS (max 50): ")
	for rows.Next() {
		err = rows.Scan(&uid, &accountID, &symbol, &priceLimit, &amount)
		println(fmt.Sprintf("UID: %s -- AccountID: %f -- Symbol: %s -- PriceLimit: %f -- Amount: %f", uid, accountID, symbol, priceLimit, amount))
	}
}

func outputSellOrders(rowLimit int) {
	tableName := "sell_order"
	var uid string
	var accountID string
	var symbol string
	var priceLimit float64
	var amount float64

	rows, err := SharedModel().db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT %d", tableName, rowLimit))
	if err != nil {
		log.Info("Error attempting to print sell orders: ", err)
		return
	}
	println("\n#######  SELL ORDERS (max 50): ")
	for rows.Next() {
		err = rows.Scan(&uid, &accountID, &symbol, &priceLimit, &amount)
		println(fmt.Sprintf("UID: %s -- AccountID: %f -- Symbol: %s -- PriceLimit: %f -- Amount: %f", uid, accountID, symbol, priceLimit, amount))
	}
}
