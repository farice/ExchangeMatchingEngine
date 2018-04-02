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
		log.Fatal("DATABASE ERROR: $1", err)
	}

	db.QueryRow(`INSERT INTO position(uid, account_id, symbol, amount)
	VALUES(1, 1, 'AMZN', 242.77)`)

	if err != nil {
		log.Error(err)
	}

}
