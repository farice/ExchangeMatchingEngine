package main

import (
	"database/sql"
	"github.com/lib/pq"
)

func init() {
	dbInfoString := "user=andrewbihl dbname=exchange sslmode=disable"
	db, err := sql.Open("postgres", dbInfoString)
	if err != nil {
		log.Fatal("DATABASE ERROR: $1", err)
	}

	
	
	
}