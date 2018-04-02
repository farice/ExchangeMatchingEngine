package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("Hello World")
	dbInfoString := "user=andrewbihl dbname=exchange sslmode=disable"
	db, err := sql.Open("postgres", dbInfoString)
	if err != nil {
		log.Fatal("DATABASE ERROR: ", err)
	}

	// db.QueryRow(`INSERT INTO position(uid, account_id, symbol, amount)
	// VALUES(1, 1, 'AMZN', 242.77)`)

	// if err != nil {
	// 	log.Error(err)
	// }

	sqlQuery := fmt.Sprintf(`SELECT shares FROM symbol WHERE symbol='%s'`, "ANDREW")
	var shares int
	fetch_err := db.QueryRow(sqlQuery).Scan(shares)
	if fetch_err != nil {
		print("ERROR: ")
		log.Error(err)
	}
	println(shares)

	model := Model{db, make(chan string, 100)}
	model.createOrUpdateSymbol("NFLX", 666)
	model.createOrUpdateSymbol("AAPL", 222)
	model.createOrUpdateSymbol("XXX", 333)
	model.executeQueries()
	println("DONE")
}
