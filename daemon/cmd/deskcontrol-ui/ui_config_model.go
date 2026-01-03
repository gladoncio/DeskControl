package main

type AppConfig struct {
	ListenIP          string
	WSPort            int
	UDPPort           int
	EncryptTrafficTLS bool

	TLSCertPath string
	TLSKeyPath  string

	// Pairing / auth
	Token        string // shared token (QR)
	RequireToken bool   // si true, WS requiere token

	RequireAccount bool   // si true, WS requiere BasicAuth (solo TLS)
	Username       string // optional
	PasswordHash   string // bcrypt hash string

	LogRetentionDays int
}

func defaultConfig() AppConfig {
	return AppConfig{
		ListenIP:          "0.0.0.0",
		WSPort:            54545,
		UDPPort:           54546,
		EncryptTrafficTLS: false,
		TLSCertPath:       "",
		TLSKeyPath:        "",
		Token:             "",
		RequireToken:      false,
		RequireAccount:    false,
		Username:          "",
		PasswordHash:      "",
		LogRetentionDays:  7,
	}
}
