package main

import (
	"database/sql"
	log "github.com/sirupsen/logrus"
//	"github.com/lib/pq"
)

func init() {
	dbInfoString := "user=andrewbihl dbname=exchange sslmode=disable"
	_, err := sql.Open("postgres", dbInfoString)
	if err != nil {
		log.Fatal("DATABASE ERROR: $1", err)
	}




}
