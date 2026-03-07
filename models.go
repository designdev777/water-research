package main

import (
    "database/sql"
    "time"
)

type Item struct {
    ID          int64
    Title       string
    Summary     string
    URL         string
    Source      string
    PublishedAt sql.NullTime
    FetchedAt   time.Time
    Notes       sql.NullString
    IsArchived  bool
}

type Proposal struct {
    ID          int64
    Title       string
    Description string
    FileName    string
    FilePath    string
    CreatedAt   time.Time
    IsActive    bool
}

func InsertItem(it *Item) error {
    stmt := `
    INSERT OR IGNORE INTO items (title, summary, url, source, published_at)
    VALUES (?, ?, ?, ?, ?)
    `
    var pub interface{}
    if it.PublishedAt.Valid {
        pub = it.PublishedAt.Time
    } else {
        pub = nil
    }
    _, err := DB.Exec(stmt, it.Title, it.Summary, it.URL, it.Source, pub)
    return err
}

func ListItems(limit int) ([]Item, error) {
    rows, err := DB.Query(`
        SELECT id, title, summary, url, source, published_at, fetched_at, notes, is_archived 
        FROM items 
        WHERE is_archived = 0 
        ORDER BY fetched_at DESC 
        LIMIT ?`, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var res []Item
    for rows.Next() {
        var it Item
        var pub sql.NullTime
        var isArchived bool
        if err := rows.Scan(&it.ID, &it.Title, &it.Summary, &it.URL, &it.Source, &pub, &it.FetchedAt, &it.Notes, &isArchived); err != nil {
            return nil, err
        }
        it.PublishedAt = pub
        it.IsArchived = isArchived
        res = append(res, it)
    }
    return res, nil
}

func GetItemByID(id int64) (*Item, error) {
    var it Item
    var pub sql.NullTime
    var notes sql.NullString
    var isArchived bool
    row := DB.QueryRow("SELECT id, title, summary, url, source, published_at, fetched_at, notes, is_archived FROM items WHERE id = ?", id)
    if err := row.Scan(&it.ID, &it.Title, &it.Summary, &it.URL, &it.Source, &pub, &it.FetchedAt, &notes, &isArchived); err != nil {
        return nil, err
    }
    it.PublishedAt = pub
    it.Notes = notes
    it.IsArchived = isArchived
    return &it, nil
}

func ArchiveOldItems(daysOld int, reason string) error {
    archiveStmt := `
    INSERT INTO archived_items 
    (original_id, title, summary, url, source, published_at, fetched_at, notes, archive_reason)
    SELECT id, title, summary, url, source, published_at, fetched_at, notes, ?
    FROM items 
    WHERE fetched_at < datetime('now', '-' || ? || ' days') 
    AND is_archived = 0
    `
    
    _, err := DB.Exec(archiveStmt, reason, daysOld)
    if err != nil {
        return err
    }
    
    updateStmt := `UPDATE items SET is_archived = 1 WHERE fetched_at < datetime('now', '-' || ? || ' days') AND is_archived = 0`
    _, err = DB.Exec(updateStmt, daysOld)
    return err
}

func GetArchivedItems(limit int) ([]Item, error) {
    rows, err := DB.Query(`
        SELECT original_id, title, summary, url, source, published_at, fetched_at, notes, archived_at 
        FROM archived_items 
        ORDER BY archived_at DESC 
        LIMIT ?`, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var res []Item
    for rows.Next() {
        var it Item
        var pub sql.NullTime
        if err := rows.Scan(&it.ID, &it.Title, &it.Summary, &it.URL, &it.Source, &pub, &it.FetchedAt, &it.Notes, &it.FetchedAt); err != nil {
            return nil, err
        }
        it.PublishedAt = pub
        it.IsArchived = true
        res = append(res, it)
    }
    return res, nil
}

// Proposal functions
func CreateProposalsTable() error {
    createStmt := `
    CREATE TABLE IF NOT EXISTS proposals (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        description TEXT,
        file_name TEXT NOT NULL,
        file_path TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        is_active BOOLEAN DEFAULT 1
    );`
    
    _, err := DB.Exec(createStmt)
    return err
}

func InsertProposal(p *Proposal) error {
    stmt := `INSERT INTO proposals (title, description, file_name, file_path) VALUES (?, ?, ?, ?)`
    _, err := DB.Exec(stmt, p.Title, p.Description, p.FileName, p.FilePath)
    return err
}

func ListProposals() ([]Proposal, error) {
    rows, err := DB.Query("SELECT id, title, description, file_name, file_path, created_at, is_active FROM proposals WHERE is_active = 1 ORDER BY created_at DESC")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var proposals []Proposal
    for rows.Next() {
        var p Proposal
        if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.FileName, &p.FilePath, &p.CreatedAt, &p.IsActive); err != nil {
            return nil, err
        }
        proposals = append(proposals, p)
    }
    return proposals, nil
}

func GetProposalByID(id int64) (*Proposal, error) {
    var p Proposal
    row := DB.QueryRow("SELECT id, title, description, file_name, file_path, created_at, is_active FROM proposals WHERE id = ? AND is_active = 1", id)
    if err := row.Scan(&p.ID, &p.Title, &p.Description, &p.FileName, &p.FilePath, &p.CreatedAt, &p.IsActive); err != nil {
        return nil, err
    }
    return &p, nil
}