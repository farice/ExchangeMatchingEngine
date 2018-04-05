package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// func logDatabaseStateTruncated() {
// 	SharedModel().executeQueries()

// }

func outputAccounts() {
	rows, err := SharedModel().db.Query("SELECT * FROM ACCOUNT LIMIT 50")
	if err != nil {
		log.Info("Error attempting to print accounts: ", err)
		return
	}
	var uid string
	var balance float64
	println("ACCOUNTS (max 50): ")
	for rows.Next() {
		err = rows.Scan(&uid, &balance)
		println(fmt.Sprintf("AccountID: %s -- Balance: %f", uid, balance))
	}
}
