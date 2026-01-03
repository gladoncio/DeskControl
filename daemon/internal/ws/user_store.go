package ws

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type userRow struct {
	Username     string
	PasswordHash string
	Disabled     bool
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
	dsn := "file:" + filepath.ToSlash(p)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// si no existe la tabla (por si alguien activa login sin crear usuarios), lo manejamos con error
	return db, nil
}

func loadUser(username string) (*userRow, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("username vacío")
	}

	db, err := openUsersDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// users table creada por tu UI
	var u userRow
	var disabledInt int

	err = db.QueryRow(`
SELECT username, password_hash, disabled
FROM users
WHERE username = ?
LIMIT 1;
`, username).Scan(&u.Username, &u.PasswordHash, &disabledInt)

	if err != nil {
		return nil, err
	}
	u.Disabled = disabledInt != 0
	return &u, nil
}

func markLastLogin(username string) {
	username = strings.TrimSpace(username)
	if username == "" {
		return
	}

	db, err := openUsersDB()
	if err != nil {
		return
	}
	defer db.Close()

	_, _ = db.Exec(`UPDATE users SET last_login_at=? WHERE username=?;`, time.Now().Unix(), username)
}
