package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite"
)

func appDataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "DeskControl")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func dbPath() (string, error) {
	dir, err := appDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "deskcontrol.db"), nil
}

func openDB() (*sql.DB, error) {
	p, err := dbPath()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", p)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func setSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(`INSERT INTO settings(key,value) VALUES(?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func getSetting(db *sql.DB, key string) (string, bool, error) {
	var v string
	err := db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func LoadConfig() (AppConfig, error) {
	cfg := defaultConfig()
	db, err := openDB()
	if err != nil {
		return cfg, err
	}
	defer db.Close()

	readStr := func(k string, dst *string) error {
		if v, ok, err := getSetting(db, k); err != nil {
			return err
		} else if ok {
			*dst = v
		}
		return nil
	}
	readInt := func(k string, dst *int) error {
		if v, ok, err := getSetting(db, k); err != nil {
			return err
		} else if ok {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
			}
		}
		return nil
	}
	readBool := func(k string, dst *bool) error {
		if v, ok, err := getSetting(db, k); err != nil {
			return err
		} else if ok {
			*dst = (v == "1" || v == "true")
		}
		return nil
	}

	_ = readStr("listen_ip", &cfg.ListenIP)
	_ = readInt("ws_port", &cfg.WSPort)
	_ = readInt("udp_port", &cfg.UDPPort)
	_ = readBool("encrypt_tls", &cfg.EncryptTrafficTLS)
	_ = readStr("tls_cert_path", &cfg.TLSCertPath)
	_ = readStr("tls_key_path", &cfg.TLSKeyPath)

	_ = readStr("token", &cfg.Token)
	_ = readBool("require_token", &cfg.RequireToken)

	_ = readBool("require_account", &cfg.RequireAccount)
	_ = readStr("username", &cfg.Username)
	_ = readStr("password_hash", &cfg.PasswordHash)

	_ = readInt("log_retention_days", &cfg.LogRetentionDays)

	// ✅ Enforce current policy on load too (so UI reflects it)
	if !cfg.EncryptTrafficTLS {
		cfg.RequireToken = false
		cfg.Token = ""
		cfg.RequireAccount = false
		cfg.Username = ""
		cfg.PasswordHash = ""
	} else {
		cfg.RequireToken = true
		if cfg.Token == "" {
			if tok, err := generateTokenURLSafe(32); err == nil {
				cfg.Token = tok
			}
		}
	}

	return cfg, nil
}

func SaveConfig(cfg AppConfig) error {
	// ---- validation ----
	if cfg.WSPort <= 0 || cfg.WSPort > 65535 {
		return fmt.Errorf("ws_port inválido: %d", cfg.WSPort)
	}
	if cfg.UDPPort <= 0 || cfg.UDPPort > 65535 {
		return fmt.Errorf("udp_port inválido: %d", cfg.UDPPort)
	}
	if cfg.LogRetentionDays < 0 {
		return fmt.Errorf("log_retention_days inválido: %d", cfg.LogRetentionDays)
	}

	// ✅ Policy: either (no TLS, no token, no account) OR (TLS + token required)
	if !cfg.EncryptTrafficTLS {
		cfg.RequireToken = false
		cfg.Token = ""
		cfg.RequireAccount = false
		cfg.Username = ""
		cfg.PasswordHash = ""
	} else {
		cfg.RequireToken = true
		if cfg.Token == "" {
			tok, err := generateTokenURLSafe(32)
			if err != nil {
				return fmt.Errorf("no pude generar token: %w", err)
			}
			cfg.Token = tok
		}
		// account is optional, but only allowed with TLS
		if cfg.RequireAccount {
			if cfg.Username == "" || cfg.PasswordHash == "" {
				return fmt.Errorf("require_account activado pero falta usuario/clave")
			}
		} else {
			// si no requiere cuenta, no obligamos limpiar user/hash; lo dejamos para que no se pierda
		}
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	write := func(k, v string) error { return setSetting(db, k, v) }
	writeInt := func(k string, v int) error { return write(k, strconv.Itoa(v)) }
	writeBool := func(k string, v bool) error {
		if v {
			return write(k, "1")
		}
		return write(k, "0")
	}

	if err := write("listen_ip", cfg.ListenIP); err != nil {
		return err
	}
	if err := writeInt("ws_port", cfg.WSPort); err != nil {
		return err
	}
	if err := writeInt("udp_port", cfg.UDPPort); err != nil {
		return err
	}

	if err := writeBool("encrypt_tls", cfg.EncryptTrafficTLS); err != nil {
		return err
	}
	if err := write("tls_cert_path", cfg.TLSCertPath); err != nil {
		return err
	}
	if err := write("tls_key_path", cfg.TLSKeyPath); err != nil {
		return err
	}

	if err := write("token", cfg.Token); err != nil {
		return err
	}
	if err := writeBool("require_token", cfg.RequireToken); err != nil {
		return err
	}

	if err := writeBool("require_account", cfg.RequireAccount); err != nil {
		return err
	}
	if err := write("username", cfg.Username); err != nil {
		return err
	}
	if err := write("password_hash", cfg.PasswordHash); err != nil {
		return err
	}

	if err := writeInt("log_retention_days", cfg.LogRetentionDays); err != nil {
		return err
	}

	return nil
}
