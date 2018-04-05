package main

import log "github.com/sirupsen/logrus"

// func logDatabaseStateTruncated() {
// 	SharedModel().executeQueries()

// }

func outputAccounts() {
	rows, err := SharedModel().db.Query("SELECT TOP 50 * FROM ACCOUNT")
	if err != nil {
		log.Info("Error attempting to print accounts: ", err)
		return
	}
	var result string
	println("ACCOUNTS (max 50): ")
	for rows.Next() {
		err = rows.Scan(&result)
		println(result)
	}
}
