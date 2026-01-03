package ws

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type SessionInfo struct {
	ID          string
	Username    string
	RemoteAddr  string
	ConnectedAt int64
	LastSeenAt  int64
	Authed      bool
}

type sessionEntry struct {
	id          string
	username    string
	remoteAddr  string
	connectedAt int64
	lastSeenAt  int64
	authed      bool

	// puntero para poder cortar
	conn *safeConn
}

var (
	sessionsMu sync.Mutex
	sessions   = map[string]*sessionEntry{}
)

func newSessionID() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func registerSession(conn *safeConn, remote string) *sessionEntry {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	id := newSessionID()
	now := time.Now().Unix()

	se := &sessionEntry{
		id:          id,
		username:    "",
		remoteAddr:  remote,
		connectedAt: now,
		lastSeenAt:  now,
		authed:      false,
		conn:        conn,
	}
	sessions[id] = se
	return se
}

func unregisterSession(id string) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	delete(sessions, id)
}

func markSessionAuthed(id, username string) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	if se, ok := sessions[id]; ok {
		se.authed = true
		se.username = username
		se.lastSeenAt = time.Now().Unix()
	}
}

func touchSession(id string) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	if se, ok := sessions[id]; ok {
		se.lastSeenAt = time.Now().Unix()
	}
}

func ListSessions() []SessionInfo {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	out := make([]SessionInfo, 0, len(sessions))
	for _, se := range sessions {
		out = append(out, SessionInfo{
			ID:          se.id,
			Username:    se.username,
			RemoteAddr:  se.remoteAddr,
			ConnectedAt: se.connectedAt,
			LastSeenAt:  se.lastSeenAt,
			Authed:      se.authed,
		})
	}
	return out
}

// DropSession corta la conexión websocket (si existe)
func DropSession(id string) bool {
	sessionsMu.Lock()
	se, ok := sessions[id]
	sessionsMu.Unlock()
	if !ok || se == nil || se.conn == nil || se.conn.c == nil {
		return false
	}
	_ = se.conn.c.Close() // esto dispara el loop de lectura y limpiará
	return true
}
