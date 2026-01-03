package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func exeDir2() string {
	p, err := os.Executable()
	if err != nil || p == "" {
		if wd, err2 := os.Getwd(); err2 == nil && wd != "" {
			return wd
		}
		return "."
	}
	return filepath.Dir(p)
}

// EnsureTLSCertKey makes sure cfg.TLSCertPath and cfg.TLSKeyPath exist.
// If they are empty or missing, it generates a self-signed cert next to the exe:
//
//	.\tls\cert.pem and .\tls\key.pem
func EnsureTLSCertKey(cfg *AppConfig) error {
	if cfg == nil || !cfg.EncryptTrafficTLS {
		return nil
	}

	// If both paths exist and files exist, ok.
	if cfg.TLSCertPath != "" && cfg.TLSKeyPath != "" {
		if fileExists(cfg.TLSCertPath) && fileExists(cfg.TLSKeyPath) {
			return nil
		}
	}

	tlsDir := filepath.Join(exeDir2(), "tls")
	if err := os.MkdirAll(tlsDir, 0o755); err != nil {
		return err
	}
	certPath := filepath.Join(tlsDir, "cert.pem")
	keyPath := filepath.Join(tlsDir, "key.pem")

	// If already generated, reuse.
	if fileExists(certPath) && fileExists(keyPath) {
		cfg.TLSCertPath = certPath
		cfg.TLSKeyPath = keyPath
		return nil
	}

	// Generate key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return err
	}

	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "DeskControl",
			Organization: []string{"DeskControl"},
		},
		NotBefore: time.Now().Add(-10 * time.Minute),
		NotAfter:  time.Now().AddDate(5, 0, 0),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true, // simplifica para pinning (autofirmado)
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}

	// Write cert
	if err := writePEM(certPath, "CERTIFICATE", der); err != nil {
		return err
	}
	// Write key
	privBytes := x509.MarshalPKCS1PrivateKey(key)
	if err := writePEM(keyPath, "RSA PRIVATE KEY", privBytes); err != nil {
		return err
	}

	cfg.TLSCertPath = certPath
	cfg.TLSKeyPath = keyPath
	return nil
}

func writePEM(path string, typ string, bytes []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: typ, Bytes: bytes})
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
