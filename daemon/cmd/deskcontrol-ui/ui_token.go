package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

func generateTokenURLSafe(nbytes int) (string, error) {
	if nbytes <= 0 {
		nbytes = 32
	}
	b := make([]byte, nbytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// bestLocalIP tries to pick a sane LAN IP (for QR host field).
func bestLocalIP() string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			// Evitar 169.254.x.x
			if ip[0] == 169 && ip[1] == 254 {
				continue
			}
			return ip.String()
		}
	}
	return "127.0.0.1"
}

// certFingerprintSHA256Hex reads a PEM cert and returns SHA-256 of its DER bytes as HEX.
func certFingerprintSHA256Hex(certPath string) (string, error) {
	certPath = strings.TrimSpace(certPath)
	if certPath == "" {
		return "", fmt.Errorf("TLSCertPath vacío")
	}
	pemBytes, err := os.ReadFile(certPath)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("cert inválido (PEM) en %s", certPath)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(cert.Raw) // DER
	return hex.EncodeToString(sum[:]), nil
}

// tokenQRPNGForConfig returns PNG bytes + payload string.
func tokenQRPNGForConfig(cfg AppConfig) ([]byte, string, error) {
	host := bestLocalIP()
	port := cfg.WSPort

	q := url.Values{}
	q.Set("host", host)
	q.Set("port", fmt.Sprintf("%d", port))

	if cfg.EncryptTrafficTLS {
		q.Set("tls", "1")

		if strings.TrimSpace(cfg.Token) != "" {
			q.Set("token", strings.TrimSpace(cfg.Token))
		} else {
			return nil, "", fmt.Errorf("TLS ON pero token está vacío")
		}

		// ✅ fingerprint en vez de cert completo (QR chico y legible)
		fp, err := certFingerprintSHA256Hex(cfg.TLSCertPath)
		if err != nil {
			return nil, "", fmt.Errorf("no pude obtener fingerprint: %w", err)
		}
		q.Set("fp", fp)
	} else {
		q.Set("tls", "0")
		// modo simple: sin token/fp
	}

	payload := "deskcontrol://pair?" + q.Encode()

	// QR pequeño y fácil de escanear
	png, err := qrcode.Encode(payload, qrcode.Medium, 320)
	return png, payload, err
}
