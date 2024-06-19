package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

var restart chan int
var passwdPath string

func init() {
	var err error
	db, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal(err)
	}

	// 创建用户表
	createUserTable := `CREATE TABLE IF NOT EXISTS users (
		username TEXT PRIMARY KEY,
		password TEXT,
		register_date TEXT,
		expiry_date TEXT,
		update_date TEXT,
		auth_code TEXT,
		register_ip TEXT
	);`

	_, err = db.Exec(createUserTable)
	if err != nil {
		log.Fatal(err)
	}

	// 创建授权码表
	createAuthCodeTable := `CREATE TABLE IF NOT EXISTS authcodes (
		code TEXT PRIMARY KEY,
		type INTEGER,
		used_by TEXT NOT NULL DEFAULT "",
		used_date TEXT
	);`

	_, err = db.Exec(createAuthCodeTable)
	if err != nil {
		log.Fatal(err)
	}

	restart = make(chan int, 100)
	passwdPath = "./passwd"
}
