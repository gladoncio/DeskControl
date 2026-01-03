package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type UserRecord struct {
	Username     string
	PasswordHash string
	Disabled     bool
	CreatedAt    int64
	LastLoginAt  int64
}

func usersDBPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	base := filepath.Dir(exe)
	return filepath.Join(base, "deskcontrol.db"), nil
}

func openUsersDB() (*sql.DB, error) {
	p, err := usersDBPath()
	if err != nil {
		return nil, err
	}

	// modernc sqlite DSN: "file:<path>"
	dsn := "file:" + filepath.ToSlash(p)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Basic pragmas (safe defaults)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON;`); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := ensureUsersSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func ensureUsersSchema(db *sql.DB) error {
	// username UNIQUE NOCASE => evita "Admin" y "admin" duplicados
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE COLLATE NOCASE,
  password_hash TEXT NOT NULL,
  disabled INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  last_login_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_users_disabled ON users(disabled);
`)
	return err
}

func LoadUsers() ([]UserRecord, error) {
	db, err := openUsersDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
SELECT username, password_hash, disabled, created_at, last_login_at
FROM users
ORDER BY LOWER(username) ASC;
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UserRecord
	for rows.Next() {
		var u UserRecord
		var disabledInt int
		if err := rows.Scan(&u.Username, &u.PasswordHash, &disabledInt, &u.CreatedAt, &u.LastLoginAt); err != nil {
			return nil, err
		}
		u.Disabled = disabledInt != 0
		out = append(out, u)
	}
	return out, rows.Err()
}

func UpsertUser(rec UserRecord) error {
	rec.Username = strings.TrimSpace(rec.Username)
	if rec.Username == "" {
		return fmt.Errorf("username requerido")
	}
	if strings.TrimSpace(rec.PasswordHash) == "" {
		return fmt.Errorf("password_hash requerido")
	}

	db, err := openUsersDB()
	if err != nil {
		return err
	}
	defer db.Close()

	now := time.Now().Unix()
	if rec.CreatedAt == 0 {
		rec.CreatedAt = now
	}

	disabledInt := 0
	if rec.Disabled {
		disabledInt = 1
	}

	// SQLite UPSERT
	_, err = db.Exec(`
INSERT INTO users(username, password_hash, disabled, created_at, last_login_at)
VALUES (?, ?, ?, ?, COALESCE(NULLIF(?,0),0))
ON CONFLICT(username) DO UPDATE SET
  password_hash=excluded.password_hash,
  disabled=excluded.disabled;
`, rec.Username, rec.PasswordHash, disabledInt, rec.CreatedAt, rec.LastLoginAt)

	return err
}

func SetUserDisabled(username string, disabled bool) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username requerido")
	}

	db, err := openUsersDB()
	if err != nil {
		return err
	}
	defer db.Close()

	disabledInt := 0
	if disabled {
		disabledInt = 1
	}

	res, err := db.Exec(`UPDATE users SET disabled=? WHERE username=?;`, disabledInt, username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("usuario no encontrado: %s", username)
	}
	return nil
}

func UpdateUserPassword(username, passwordHash string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username requerido")
	}
	if strings.TrimSpace(passwordHash) == "" {
		return fmt.Errorf("password_hash requerido")
	}

	db, err := openUsersDB()
	if err != nil {
		return err
	}
	defer db.Close()

	res, err := db.Exec(`UPDATE users SET password_hash=? WHERE username=?;`, passwordHash, username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("usuario no encontrado: %s", username)
	}
	return nil
}

func DeleteUser(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username requerido")
	}

	db, err := openUsersDB()
	if err != nil {
		return err
	}
	defer db.Close()

	res, err := db.Exec(`DELETE FROM users WHERE username=?;`, username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("usuario no encontrado: %s", username)
	}
	return nil
}
