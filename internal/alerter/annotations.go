package alerter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// AnnotationConfig holds Grafana annotation settings.
type AnnotationConfig struct {
	GrafanaURL   string // e.g. http://grafana:3000
	GrafanaToken string // API key or service account token
}

// Annotator pushes annotations to Grafana on anomaly spikes.
type Annotator struct {
	cfg    AnnotationConfig
	client *http.Client
	alerts *Alerter // reuse cooldown tracking
}

// NewAnnotator creates a Grafana annotator. Returns nil if not configured.
func NewAnnotator(cfg AnnotationConfig, cooldown time.Duration) *Annotator {
	if cfg.GrafanaURL == "" || cfg.GrafanaToken == "" {
		return nil
	}
	return &Annotator{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
		alerts: New(Config{
			WebhookURL: "noop", // enables the Alerter for cooldown tracking
			Cooldown:   cooldown,
		}),
	}
}

// Annotate pushes a Grafana annotation if not in cooldown.
func (a *Annotator) Annotate(alertType AlertType, host, text string, tags []string) {
	if a == nil {
		return
	}

	// Use alerter's cooldown mechanism.
	key := string(alertType) + ":" + host
	a.alerts.mu.Lock()
	if last, ok := a.alerts.lastSent[key]; ok && time.Since(last) < a.alerts.cfg.Cooldown {
		a.alerts.mu.Unlock()
		return
	}
	a.alerts.lastSent[key] = time.Now()
	a.alerts.mu.Unlock()

	annotation := map[string]interface{}{
		"text": fmt.Sprintf("[%s] %s: %s", host, alertType, text),
		"tags": append(tags, "mysqlpulse", string(alertType), host),
	}

	body, _ := json.Marshal(annotation)
	req, err := http.NewRequest("POST", a.cfg.GrafanaURL+"/api/annotations", bytes.NewReader(body))
	if err != nil {
		log.Printf("grafana annotation error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.GrafanaToken)

	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("grafana annotation error: %v", err)
		return
	}
	_ = resp.Body.Close()
}
