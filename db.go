package main

import (
    "database/sql"
    "log"

    _ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(path string) error {
    var err error
    DB, err = sql.Open("sqlite3", path)
    if err != nil {
        return err
    }

    if err := DB.Ping(); err != nil {
        return err
    }

    createStmt := `
    CREATE TABLE IF NOT EXISTS items (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT,
        summary TEXT,
        url TEXT UNIQUE,
        source TEXT,
        published_at DATETIME,
        fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        notes TEXT,
        is_archived BOOLEAN DEFAULT 0
    );
    
    CREATE TABLE IF NOT EXISTS archived_items (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        original_id INTEGER,
        title TEXT,
        summary TEXT,
        url TEXT,
        source TEXT,
        published_at DATETIME,
        fetched_at DATETIME,
        notes TEXT,
        archived_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        archive_reason TEXT
    );`
    
    if _, err := DB.Exec(createStmt); err != nil {
        return err
    }

    log.Println("database initialized")
    return nil
}