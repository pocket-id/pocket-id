package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

type WebhookService struct {
	appConfigService *AppConfigService
	httpClient       *http.Client
}

func NewWebhookService(appConfigService *AppConfigService, httpClient *http.Client) *WebhookService {
	return &WebhookService{
		appConfigService: appConfigService,
		httpClient:       httpClient,
	}
}

// Discord/Slack embed payload types

type webhookEmbed struct {
	Title     string              `json:"title"`
	Color     int                 `json:"color"`
	Fields    []webhookEmbedField `json:"fields"`
	Timestamp string              `json:"timestamp"`
}

type webhookEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type webhookPayload struct {
	Content     string         `json:"content,omitempty"`
	Embeds      []webhookEmbed `json:"embeds,omitempty"`      // Discord
	Attachments []webhookEmbed `json:"attachments,omitempty"` // Slack
}

// SendEvent sends a webhook notification for an audit log event.
// It checks if the webhook URL is configured and the event matches the filter.
// This method is designed to be called asynchronously in a goroutine.
func (s *WebhookService) SendEvent(ctx context.Context, auditLog model.AuditLog) {
	cfg := s.appConfigService.GetDbConfig()
	webhookURL := cfg.WebhookUrl.Value
	if webhookURL == "" {
		return
	}

	// Check if the event matches the configured filter
	if !s.isEventAllowed(string(auditLog.Event)) {
		return
	}

	ipAddress := ""
	if auditLog.IpAddress != nil {
		ipAddress = *auditLog.IpAddress
	}

	location := formatLocation(auditLog.Country, auditLog.City)

	payload := webhookPayload{
		Embeds: []webhookEmbed{
			{
				Title: formatEventTitle(string(auditLog.Event)),
				Color: 5814783, // A pleasant blue/purple color
				Fields: []webhookEmbedField{
					{Name: "User", Value: valueOrDash(auditLog.Username), Inline: true},
					{Name: "IP Address", Value: valueOrDash(ipAddress), Inline: true},
					{Name: "Location", Value: valueOrDash(location), Inline: true},
					{Name: "Device", Value: valueOrDash(auditLog.UserAgent), Inline: true},
				},
				Timestamp: auditLog.CreatedAt.UTC().Format(time.RFC3339),
			},
		},
	}

	// Add any extra data fields from the audit log.
	for k, v := range auditLog.Data {
		payload.Embeds[0].Fields = append(payload.Embeds[0].Fields, webhookEmbedField{
			Name:   k,
			Value:  valueOrDash(v),
			Inline: true,
		})
	}

	err := s.sendPayload(ctx, webhookURL, payload)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to send webhook", slog.Any("error", err), slog.String("event", string(auditLog.Event)))
	}
}

// SendTestWebhook sends a test webhook to verify connectivity.
func (s *WebhookService) SendTestWebhook(ctx context.Context) error {
	cfg := s.appConfigService.GetDbConfig()
	webhookURL := cfg.WebhookUrl.Value
	if webhookURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}

	payload := webhookPayload{
		Embeds: []webhookEmbed{
			{
				Title: "Test Webhook",
				Color: 5814783,
				Fields: []webhookEmbedField{
					{Name: "Status", Value: "Connection successful", Inline: false},
					{Name: "Source", Value: "Pocket ID", Inline: true},
				},
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	return s.sendPayload(ctx, webhookURL, payload)
}

func (s *WebhookService) sendPayload(ctx context.Context, webhookURL string, payload webhookPayload) error {
	var finalPayload any
	if strings.Contains(webhookURL, "hooks.slack.com") {
		payload.Attachments = payload.Embeds
		payload.Embeds = nil
		finalPayload = payload
	} else {
		finalPayload = payload
	}

	body, err := json.Marshal(finalPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d", resp.StatusCode)
	}

	return nil
}

// isEventAllowed checks if the given event is in the configured event filter.
// If no filter is configured (empty string), all events are allowed.
func (s *WebhookService) isEventAllowed(event string) bool {
	cfg := s.appConfigService.GetDbConfig()
	eventsFilter := cfg.WebhookEvents.Value
	if eventsFilter == "" {
		return true
	}

	for _, allowed := range strings.Split(eventsFilter, ",") {
		if strings.TrimSpace(allowed) == event {
			return true
		}
	}

	return false
}

// formatEventTitle converts an event constant like "SIGN_IN" to a title like "Sign In"
func formatEventTitle(event string) string {
	words := strings.Split(event, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func formatLocation(country, city string) string {
	if country == "" && city == "" {
		return ""
	}
	if city == "" {
		return country
	}
	if country == "" {
		return city
	}
	return city + ", " + country
}

func valueOrDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}
