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
}

// StartUDP starts a UDP discovery responder.
// It listens on udpPort (e.g. 54546) and answers "discover" requests with "announce".
func StartUDP(name string, wsPort int, udpPort int) {
	addr := net.UDPAddr{IP: net.IPv4zero, Port: udpPort}
	conn, err := net.ListenUDP("udp4", &addr)
	if err != nil {
		log.Println("discovery udp listen error:", err)
		return
	}
	defer conn.Close()

	_ = conn.SetReadBuffer(1 << 20)

	log.Println("Discovery UDP listening on", conn.LocalAddr().String())

	buf := make([]byte, 2048)

	for {
		_ = conn.SetReadDeadline(time.Time{})

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
		}

		b, _ := json.Marshal(resp)
		if _, err := conn.WriteToUDP(b, remote); err != nil {
			log.Println("discovery write error:", err)
		}

	}
}
