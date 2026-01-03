package ws

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"deskcontrol/daemon/internal/input"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
)

type SecurityConfig struct {
	RequireTLS bool
	CertPath   string
	KeyPath    string

	RequireToken bool
	Token        string

	RequireAccount bool // SOLO con TLS
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type safeConn struct {
	c  *websocket.Conn
	mu sync.Mutex // gorilla/websocket: un solo writer a la vez
}

func (s *safeConn) writeJSON(v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.c.SetWriteDeadline(time.Now().Add(3 * time.Second))
	return s.c.WriteJSON(v)
}

// ---- Incoming messages ----
type baseMsg struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
}

type authLoginMsg struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type mouseMoveMsg struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
	Dx   int32  `json:"dx"`
	Dy   int32  `json:"dy"`
}

type mouseClickMsg struct {
	ID     string `json:"id,omitempty"`
	Type   string `json:"type"`
	Button string `json:"button"`
}

type mouseScrollMsg struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
	Dy   int32  `json:"dy"`
}

type keyTextMsg struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
	Text string `json:"text"`
}

type keyMsg struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
	Key  string `json:"key"`
}

type hotkeyMsg struct {
	ID   string   `json:"id,omitempty"`
	Type string   `json:"type"`
	Mods []string `json:"mods"`
	Key  string   `json:"key"`
}

type keyVKMsg struct {
	ID   string        `json:"id,omitempty"`
	Type string        `json:"type"`
	Key  input.KeySpec `json:"key"`
}

type hotkeyVKMsg struct {
	ID   string        `json:"id,omitempty"`
	Type string        `json:"type"`
	Mods []string      `json:"mods"`
	Key  input.KeySpec `json:"key"`
}

type captureStartMsg struct {
	ID        string `json:"id,omitempty"`
	Type      string `json:"type"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

type appsListMsg struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
}

type appActionMsg struct {
	ID     string  `json:"id,omitempty"`
	Type   string  `json:"type"`
	Hwnd   uintptr `json:"hwnd"`
	Action string  `json:"action"`
}

// ✅ NUEVOS (compatibles root y payload)
type inputKeyFlatMsg struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
	VK   uint16 `json:"vk"`
	Scan uint16 `json:"scan,omitempty"`
	Ext  bool   `json:"ext,omitempty"`

	Payload *struct {
		VK   uint16 `json:"vk"`
		Scan uint16 `json:"scan,omitempty"`
		Ext  bool   `json:"ext,omitempty"`
	} `json:"payload,omitempty"`
}

type textInputMsg struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
	Text string `json:"text"`

	Payload *struct {
		Text string `json:"text"`
	} `json:"payload,omitempty"`
}

func parseKeySpec(raw []byte) (input.KeySpec, bool) {
	var m inputKeyFlatMsg
	if err := json.Unmarshal(raw, &m); err != nil {
		return input.KeySpec{}, false
	}
	if m.VK != 0 || m.Scan != 0 {
		return input.KeySpec{VK: m.VK, Scan: m.Scan, Ext: m.Ext}, true
	}
	if m.Payload != nil {
		return input.KeySpec{VK: m.Payload.VK, Scan: m.Payload.Scan, Ext: m.Payload.Ext}, true
	}
	return input.KeySpec{}, true
}

func parseText(raw []byte) (string, bool) {
	var m textInputMsg
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", false
	}
	if m.Text != "" {
		return m.Text, true
	}
	if m.Payload != nil {
		return m.Payload.Text, true
	}
	return "", true
}

// ---- Outgoing ----
type errResp struct {
	ID    string `json:"id,omitempty"`
	Type  string `json:"type"`
	Error string `json:"error"`
}

type authOkResp struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type"`
	Username string `json:"username"`
	Session  string `json:"session"`
}

type captureResp struct {
	ID     string              `json:"id,omitempty"`
	Type   string              `json:"type"`
	Result input.CaptureResult `json:"result"`
}

type appsListResp struct {
	ID   string          `json:"id,omitempty"`
	Type string          `json:"type"`
	Apps []input.AppInfo `json:"apps"`
}

type pongResp struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
}

func tokenFromRequest(r *http.Request) string {
	if t := r.Header.Get("X-DeskControl-Token"); t != "" {
		return t
	}
	u, err := url.Parse(r.URL.String())
	if err == nil {
		if q := u.Query().Get("token"); q != "" {
			return q
		}
	}
	return ""
}

func checkToken(sec SecurityConfig, r *http.Request) bool {
	// Política: token solo tiene sentido con TLS
	if !sec.RequireTLS {
		return true
	}
	if !sec.RequireToken {
		return true
	}
	if sec.Token == "" {
		return false
	}
	got := tokenFromRequest(r)
	return subtle.ConstantTimeCompare([]byte(got), []byte(sec.Token)) == 1
}

func requireAccountActive(sec SecurityConfig) bool {
	// cuenta solo con TLS
	if !sec.RequireTLS {
		return false
	}
	return sec.RequireAccount
}

func Start(addr string, driver input.InputDriver, sec SecurityConfig) {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Gates BEFORE upgrade
		if !checkToken(sec, r) {
			http.Error(w, "unauthorized (token)", http.StatusUnauthorized)
			return
		}

		rawConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("[ws] upgrade error:", err)
			return
		}
		defer rawConn.Close()

		conn := &safeConn{c: rawConn}
		log.Println("[ws] client connected from", r.RemoteAddr)

		// Register session slot (even before auth) so UI can see connections
		se := registerSession(conn, r.RemoteAddr)
		sessionID := se.id

		authed := false
		username := ""

		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[ws] PANIC in handler: %v\n%s", rec, string(debug.Stack()))
			}
			unregisterSession(sessionID)
			log.Println("[ws] client disconnected:", r.RemoteAddr)
		}()

		for {
			_, raw, err := rawConn.ReadMessage()
			if err != nil {
				log.Println("[ws] read error:", err)
				return
			}

			touchSession(sessionID)

			var b baseMsg
			if err := json.Unmarshal(raw, &b); err != nil {
				log.Println("[ws] invalid json:", err)
				continue
			}

			log.Printf("[ws] recv type=%s id=%s bytes=%d", b.Type, b.ID, len(raw))

			// ✅ siempre responder ping
			if b.Type == "ping" {
				if err := conn.writeJSON(pongResp{ID: b.ID, Type: "pong"}); err != nil {
					log.Printf("[ws] pong write error: %v", err)
					return
				}
				continue
			}

			// --- LOGIN gating (solo TLS + RequireAccount) ---
			if requireAccountActive(sec) && !authed {
				if b.Type != "auth_login" {
					_ = conn.writeJSON(errResp{ID: b.ID, Type: "error", Error: "unauthorized: login requerido"})
					continue
				}

				var m authLoginMsg
				if err := json.Unmarshal(raw, &m); err != nil {
					_ = conn.writeJSON(errResp{ID: b.ID, Type: "error", Error: "auth_login inválido"})
					continue
				}
				u := strings.TrimSpace(m.Username)
				p := m.Password

				if u == "" || p == "" {
					_ = conn.writeJSON(errResp{ID: b.ID, Type: "error", Error: "usuario/contraseña requeridos"})
					continue
				}

				row, err := loadUser(u)
				if err != nil {
					// sqlite: si tabla no existe, devuelve error: treat as no users
					if err == sql.ErrNoRows {
						_ = conn.writeJSON(errResp{ID: b.ID, Type: "error", Error: "usuario o contraseña inválidos"})
						continue
					}
					// también cubre "no such table: users"
					log.Printf("[auth] loadUser error: %v", err)
					_ = conn.writeJSON(errResp{ID: b.ID, Type: "error", Error: "no hay usuarios configurados (crea uno en la UI)"})
					continue
				}

				if row.Disabled {
					_ = conn.writeJSON(errResp{ID: b.ID, Type: "error", Error: "usuario deshabilitado"})
					continue
				}

				if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(p)) != nil {
					_ = conn.writeJSON(errResp{ID: b.ID, Type: "error", Error: "usuario o contraseña inválidos"})
					continue
				}

				authed = true
				username = row.Username
				markSessionAuthed(sessionID, username)
				markLastLogin(username)

				if err := conn.writeJSON(authOkResp{ID: b.ID, Type: "auth_ok", Username: username, Session: sessionID}); err != nil {
					log.Printf("[auth] write auth_ok error: %v", err)
					return
				}
				continue
			}

			// ---- Normal actions ----
			switch b.Type {

			case "mouse_move":
				var m mouseMoveMsg
				if json.Unmarshal(raw, &m) == nil {
					_ = driver.MoveMouse(m.Dx, m.Dy)
				}

			case "mouse_click":
				var m mouseClickMsg
				if json.Unmarshal(raw, &m) == nil {
					_ = driver.MouseClick(m.Button)
				}

			case "mouse_down":
				var m mouseClickMsg
				if json.Unmarshal(raw, &m) == nil {
					_ = driver.MouseDown(m.Button)
				}

			case "mouse_up":
				var m mouseClickMsg
				if json.Unmarshal(raw, &m) == nil {
					_ = driver.MouseUp(m.Button)
				}

			case "mouse_scroll":
				var m mouseScrollMsg
				if json.Unmarshal(raw, &m) == nil {
					_ = driver.MouseScroll(m.Dy)
				}

			case "key_text":
				var m keyTextMsg
				if json.Unmarshal(raw, &m) == nil {
					log.Printf("[input] key_text text=%q", m.Text)
					_ = driver.KeyText(m.Text)
				}

			case "key":
				var m keyMsg
				if json.Unmarshal(raw, &m) == nil {
					log.Printf("[input] key key=%q", m.Key)
					_ = driver.Key(m.Key)
				}

			case "key_down":
				var m keyMsg
				if json.Unmarshal(raw, &m) == nil {
					log.Printf("[input] key_down key=%q", m.Key)
					_ = driver.KeyDown(m.Key)
				}

			case "key_up":
				var m keyMsg
				if json.Unmarshal(raw, &m) == nil {
					log.Printf("[input] key_up key=%q", m.Key)
					_ = driver.KeyUp(m.Key)
				}

			case "hotkey":
				var m hotkeyMsg
				if json.Unmarshal(raw, &m) == nil {
					log.Printf("[input] hotkey mods=%v key=%q", m.Mods, m.Key)
					_ = driver.Hotkey(m.Mods, m.Key)
				}

			case "key_vk":
				var m keyVKMsg
				if json.Unmarshal(raw, &m) == nil {
					log.Printf("[input] key_vk vk=%d scan=%d ext=%v", m.Key.VK, m.Key.Scan, m.Key.Ext)
					_ = driver.KeyVK(m.Key)
				}
			case "key_down_vk":
				var m keyVKMsg
				if json.Unmarshal(raw, &m) == nil {
					log.Printf("[input] key_down_vk vk=%d scan=%d ext=%v", m.Key.VK, m.Key.Scan, m.Key.Ext)
					_ = driver.KeyDownVK(m.Key)
				}
			case "key_up_vk":
				var m keyVKMsg
				if json.Unmarshal(raw, &m) == nil {
					log.Printf("[input] key_up_vk vk=%d scan=%d ext=%v", m.Key.VK, m.Key.Scan, m.Key.Ext)
					_ = driver.KeyUpVK(m.Key)
				}
			case "hotkey_vk":
				var m hotkeyVKMsg
				if json.Unmarshal(raw, &m) == nil {
					log.Printf("[input] hotkey_vk mods=%v vk=%d scan=%d ext=%v", m.Mods, m.Key.VK, m.Key.Scan, m.Key.Ext)
					_ = driver.HotkeyVK(m.Mods, m.Key)
				}

			case "text_input":
				if text, ok := parseText(raw); ok {
					log.Printf("[input] text_input id=%s text=%q", b.ID, text)
					if text != "" {
						_ = driver.KeyText(text)
					}
				}

			case "input_key_tap":
				if ks, ok := parseKeySpec(raw); ok {
					log.Printf("[input] input_key_tap id=%s vk=%d scan=%d ext=%v", b.ID, ks.VK, ks.Scan, ks.Ext)
					if ks.VK != 0 || ks.Scan != 0 {
						_ = driver.KeyVK(ks)
					}
				}

			case "input_key_down":
				if ks, ok := parseKeySpec(raw); ok {
					log.Printf("[input] input_key_down id=%s vk=%d scan=%d ext=%v", b.ID, ks.VK, ks.Scan, ks.Ext)
					if ks.VK != 0 || ks.Scan != 0 {
						_ = driver.KeyDownVK(ks)
					}
				}

			case "input_key_up":
				if ks, ok := parseKeySpec(raw); ok {
					log.Printf("[input] input_key_up id=%s vk=%d scan=%d ext=%v", b.ID, ks.VK, ks.Scan, ks.Ext)
					if ks.VK != 0 || ks.Scan != 0 {
						_ = driver.KeyUpVK(ks)
					}
				}

			case "capture_start":
				var m captureStartMsg
				if json.Unmarshal(raw, &m) != nil {
					continue
				}
				timeout := m.TimeoutMs
				if timeout <= 0 {
					timeout = 10000
				}

				log.Printf("[capture] start id=%s timeoutMs=%d", m.ID, timeout)

				go func(reqID string, t int) {
					defer func() {
						if rec := recover(); rec != nil {
							log.Printf("[capture] PANIC id=%s: %v\n%s", reqID, rec, string(debug.Stack()))
							_ = conn.writeJSON(errResp{ID: reqID, Type: "error", Error: "panic in capture (check daemon logs)"})
						}
					}()

					res, err := driver.CaptureNextKey(t)
					if err != nil {
						log.Printf("[capture] error id=%s: %v", reqID, err)
						_ = conn.writeJSON(errResp{ID: reqID, Type: "error", Error: err.Error()})
						return
					}

					if err := conn.writeJSON(captureResp{ID: reqID, Type: "capture_key", Result: res}); err != nil {
						log.Printf("[capture] writeJSON failed id=%s: %v", reqID, err)
					} else {
						log.Printf("[capture] response sent id=%s", reqID)
					}
				}(m.ID, timeout)

			case "apps_list":
				var m appsListMsg
				if json.Unmarshal(raw, &m) != nil {
					continue
				}
				apps, err := driver.ListApps()
				if err != nil {
					log.Printf("[apps] list error id=%s: %v", m.ID, err)
					_ = conn.writeJSON(errResp{ID: m.ID, Type: "error", Error: err.Error()})
					continue
				}
				log.Printf("[apps] list ok id=%s count=%d", m.ID, len(apps))
				_ = conn.writeJSON(appsListResp{ID: m.ID, Type: "apps_list_result", Apps: apps})

			case "app_action":
				var m appActionMsg
				if json.Unmarshal(raw, &m) != nil {
					continue
				}
				log.Printf("[apps] action id=%s hwnd=%d action=%s", m.ID, m.Hwnd, m.Action)
				if err := driver.AppAction(m.Hwnd, m.Action); err != nil {
					log.Printf("[apps] action error id=%s: %v", m.ID, err)
					_ = conn.writeJSON(errResp{ID: m.ID, Type: "error", Error: err.Error()})
				}

			default:
				// ignore
			}
		}
	})

	if sec.RequireTLS {
		log.Println("[ws] TLS ENABLED: only wss:// is allowed (ws:// will NOT be served)")
		log.Printf("[ws] Daemon listening (TLS) on %s endpoint /ws cert=%q key=%q token=%v account=%v",
			addr, sec.CertPath, sec.KeyPath, sec.RequireToken, sec.RequireAccount)
		log.Fatal(http.ListenAndServeTLS(addr, sec.CertPath, sec.KeyPath, mux))
		return
	}

	log.Println("[ws] TLS disabled: serving ws:// on", addr, "endpoint /ws")
	log.Fatal(http.ListenAndServe(addr, mux))
}
