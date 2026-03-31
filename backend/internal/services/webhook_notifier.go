package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// WebhookNotifier sends notifications via webhook
type WebhookNotifier struct {
	url     string
	secret  string
	enabled bool
	client  *http.Client
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(url, secret string) *WebhookNotifier {
	return &WebhookNotifier{
		url:     url,
		secret:  secret,
		enabled: url != "",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// WebhookPayload represents the webhook payload
type WebhookPayload struct {
	EventType string                 `json:"event_type"`
	Severity  string                 `json:"severity"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Timestamp string                 `json:"timestamp"`
	PanelURL  string                 `json:"panel_url,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Send sends a notification via webhook
func (w *WebhookNotifier) Send(notification *models.Notification) error {
	if !w.enabled || w.url == "" {
		return nil
	}

	// Parse metadata
	var metadata map[string]interface{}
	if notification.Metadata != "" {
		if err := json.Unmarshal([]byte(notification.Metadata), &metadata); err != nil {
			return fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	// Create payload
	payload := WebhookPayload{
		EventType: string(notification.EventType),
		Severity:  string(notification.Severity),
		Title:     notification.Title,
		Message:   notification.Message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Metadata:  metadata,
	}

	// Marshal payload
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", w.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Isolate-Panel-Notifications/1.0")

	// Add HMAC signature
	if w.secret != "" {
		signature := w.generateSignature(body)
		req.Header.Set("X-Isolate-Panel-Signature", signature)
	}

	// Send request with timeout
	client := w.client
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// generateSignature generates HMAC-SHA256 signature
func (w *WebhookNotifier) generateSignature(body []byte) string {
	h := hmac.New(sha256.New, []byte(w.secret))
	h.Write(body)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
