package discovery

import (
	"encoding/json"
	"log"
	"net"
	"strings"
	"time"
)

type discoverMsg struct {
	Type string `json:"type"`
	App  string `json:"app"`
	V    int    `json:"v"`
}

type announceMsg struct {
	Type   string `json:"type"`
	App    string `json:"app"`
	V      int    `json:"v"`
	Name   string `json:"name"`
	WsPort int    `json:"ws_port"`
	TLS    bool   `json:"tls,omitempty"`
}

// StartUDP listens on udpPort and answers discovery requests.
// listenIP binds the UDP socket. tlsEnabled is announced to clients.
func StartUDP(name string, wsPort int, udpPort int, listenIP string, tlsEnabled bool) {
	bindIP := net.IPv4zero
	if listenIP != "" {
		if ip := net.ParseIP(listenIP); ip != nil {
			bindIP = ip
		} else {
			log.Printf("discovery: invalid listenIP %q, using 0.0.0.0", listenIP)
		}
	}

	addr := net.UDPAddr{IP: bindIP, Port: udpPort}
	conn, err := net.ListenUDP("udp4", &addr)
	if err != nil {
		log.Println("discovery udp listen error:", err)
		return
	}
	defer conn.Close()

	log.Println("Discovery UDP listening on", conn.LocalAddr().String())

	buf := make([]byte, 2048)

	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("discovery read error:", err)
			continue
		}

		var msg discoverMsg
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}

		if strings.ToLower(msg.Type) != "discover" || strings.ToLower(msg.App) != "deskcontrol" {
			continue
		}

		resp := announceMsg{
			Type:   "announce",
			App:    "deskcontrol",
			V:      1,
			Name:   name,
			WsPort: wsPort,
			TLS:    tlsEnabled,
		}

		b, _ := json.Marshal(resp)
		_ = conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
		if _, err := conn.WriteToUDP(b, remote); err != nil {
			log.Println("discovery write error:", err)
		}
	}
}
