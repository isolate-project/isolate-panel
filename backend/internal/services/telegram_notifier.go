package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// TelegramNotifier sends notifications via Telegram Bot API
type TelegramNotifier struct {
	botToken string
	chatID   string
	enabled  bool
	client   *http.Client
}

// NewTelegramNotifier creates a new Telegram notifier
func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		enabled:  botToken != "" && chatID != "",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// TelegramMessage represents a Telegram message
type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// Send sends a notification via Telegram
func (t *TelegramNotifier) Send(notification *models.Notification) error {
	if !t.enabled || t.botToken == "" || t.chatID == "" {
		return nil
	}

	// Format message with emoji based on severity
	emoji := t.getSeverityEmoji(notification.Severity)
	text := t.formatMessage(notification, emoji)

	// Escape markdown
	text = t.escapeMarkdown(text)

	// Create message
	message := TelegramMessage{
		ChatID:    t.chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	// Marshal payload
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create request
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := t.client
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var telegramResp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.OK {
		return fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	return nil
}

// getSeverityEmoji returns emoji for severity
func (t *TelegramNotifier) getSeverityEmoji(severity models.NotificationSeverity) string {
	switch severity {
	case models.NotificationSeverityCritical:
		return "🚨"
	case models.NotificationSeverityError:
		return "❌"
	case models.NotificationSeverityWarning:
		return "⚠️"
	case models.NotificationSeverityInfo:
		return "ℹ️"
	default:
		return "📢"
	}
}

// formatMessage formats notification for Telegram
func (t *TelegramNotifier) formatMessage(notification *models.Notification, emoji string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*%s %s*\n\n", emoji, notification.Title))
	sb.WriteString(notification.Message)

	// Add metadata if available
	if notification.Metadata != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(notification.Metadata), &metadata); err == nil {
			sb.WriteString("\n\n")
			sb.WriteString("*Details:*\n")
			for key, value := range metadata {
				sb.WriteString(fmt.Sprintf("• %s: %v\n", t.formatKey(key), value))
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\n_Time: %s_", time.Now().UTC().Format("2006-01-02 15:04:05 UTC")))

	return sb.String()
}

// formatKey formats metadata key for display
func (t *TelegramNotifier) formatKey(key string) string {
	// Convert snake_case to Title Case
	parts := strings.Split(key, "_")
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}
	return strings.Join(parts, " ")
}

// escapeMarkdown escapes special markdown characters
func (t *TelegramNotifier) escapeMarkdown(text string) string {
	// Escape characters that have special meaning in Markdown v1
	replacements := map[string]string{
		"_": "\\_",
		"*": "\\*",
		"`": "\\`",
		"[": "\\[",
	}

	result := text
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	return result
}

// TestConnection tests the Telegram bot connection
func (t *TelegramNotifier) TestConnection() error {
	if !t.enabled || t.botToken == "" || t.chatID == "" {
		return fmt.Errorf("telegram not configured")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", t.botToken)
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	var telegramResp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		Result      struct {
			Username string `json:"username"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !telegramResp.OK {
		return fmt.Errorf("telegram API error: %s", telegramResp.Description)
	}

	return nil
}

// SendTestMessage sends a test message
func (t *TelegramNotifier) SendTestMessage() error {
	notification := &models.Notification{
		EventType: "test",
		Severity:  models.NotificationSeverityInfo,
		Title:     "Test Notification",
		Message:   "This is a test notification from Isolate Panel",
	}

	return t.Send(notification)
}
