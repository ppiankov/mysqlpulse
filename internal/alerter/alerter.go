package alerter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AlertType classifies the alert.
type AlertType string

const (
	AlertReplStopped    AlertType = "repl_stopped"
	AlertReplLag        AlertType = "repl_lag"
	AlertBufferPool     AlertType = "buffer_pool_pressure"
	AlertConnExhaustion AlertType = "conn_exhaustion"
	AlertDeadlocks      AlertType = "deadlocks"
	AlertHistoryList    AlertType = "history_list"
)

// Alert is a single alert event.
type Alert struct {
	Type     AlertType
	Message  string
	Instance string
	Host     string
}

// Config holds alerter configuration.
type Config struct {
	TelegramToken string
	TelegramChat  string
	WebhookURL    string
	Cooldown      time.Duration
}

// Alerter sends notifications via configured channels.
type Alerter struct {
	cfg      Config
	client   *http.Client
	mu       sync.Mutex
	lastSent map[string]time.Time // key: type+instance → last sent time
}

// New creates an Alerter. Returns nil if no channels configured.
func New(cfg Config) *Alerter {
	if cfg.TelegramToken == "" && cfg.WebhookURL == "" {
		return nil
	}
	if cfg.Cooldown == 0 {
		cfg.Cooldown = 5 * time.Minute
	}
	return &Alerter{
		cfg:      cfg,
		client:   &http.Client{Timeout: 10 * time.Second},
		lastSent: make(map[string]time.Time),
	}
}

// Send dispatches an alert if not in cooldown.
func (a *Alerter) Send(alert Alert) {
	if a == nil {
		return
	}

	a.mu.Lock()
	key := string(alert.Type) + ":" + alert.Instance
	if last, ok := a.lastSent[key]; ok && time.Since(last) < a.cfg.Cooldown {
		a.mu.Unlock()
		return
	}
	a.lastSent[key] = time.Now()
	a.mu.Unlock()

	if a.cfg.TelegramToken != "" && a.cfg.TelegramChat != "" {
		a.sendTelegram(alert)
	}
	if a.cfg.WebhookURL != "" {
		a.sendWebhook(alert)
	}
}

func (a *Alerter) sendTelegram(alert Alert) {
	text := fmt.Sprintf("<b>mysqlpulse [%s]: %s</b>\n\n%s\n\n<i>DSN: %s</i>",
		alert.Host, alert.Type, alert.Message, alert.Instance)

	body, _ := json.Marshal(map[string]string{
		"chat_id":    a.cfg.TelegramChat,
		"text":       text,
		"parse_mode": "HTML",
	})

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", a.cfg.TelegramToken)
	resp, err := a.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("telegram alert error: %v", err)
		return
	}
	_ = resp.Body.Close()
}

func (a *Alerter) sendWebhook(alert Alert) {
	payload, _ := json.Marshal(map[string]string{
		"text":     fmt.Sprintf("mysqlpulse [%s] alert [%s]: %s", alert.Host, alert.Type, alert.Message),
		"type":     string(alert.Type),
		"message":  alert.Message,
		"instance": alert.Instance,
		"host":     alert.Host,
	})

	resp, err := a.client.Post(a.cfg.WebhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("webhook alert error: %v", err)
		return
	}
	_ = resp.Body.Close()
}

// MaskDSN redacts credentials from a MySQL DSN.
// "user:pass@tcp(host:3306)/db" → "u***r:p***s@tcp(host:3306)/db"
func MaskDSN(dsn string) string {
	atIdx := strings.LastIndex(dsn, "@")
	if atIdx < 0 {
		return dsn
	}

	userInfo := dsn[:atIdx]
	rest := dsn[atIdx:]

	parts := strings.SplitN(userInfo, ":", 2)
	masked := maskMiddle(parts[0])
	if len(parts) > 1 {
		masked += ":" + maskMiddle(parts[1])
	}

	return masked + rest
}

// HostFromDSN extracts the host (with port) from a MySQL DSN.
// "user:pass@tcp(host:3306)/db" → "host:3306"
func HostFromDSN(dsn string) string {
	if i := strings.Index(dsn, "tcp("); i >= 0 {
		rest := dsn[i+4:]
		if j := strings.Index(rest, ")"); j >= 0 {
			return rest[:j]
		}
	}
	return "unknown"
}

// maskMiddle keeps the first and last character, replacing the middle with ***.
func maskMiddle(s string) string {
	if len(s) < 3 {
		return "***"
	}
	return string(s[0]) + "***" + string(s[len(s)-1])
}
