// Package sms provides an SMS sending abstraction: a console/log sender for
// development and a Kavenegar HTTP adapter for production (research.md R4).
package sms

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SmsSender sends a message to a single recipient phone number.
type SmsSender interface {
	Send(phone, message string) error
}

// ConsoleSender logs the message instead of sending it — development only.
type ConsoleSender struct {
	Log *slog.Logger
}

// Send logs the message; no real SMS is dispatched.
func (s *ConsoleSender) Send(phone, message string) error {
	log := s.Log
	if log == nil {
		log = slog.Default()
	}
	log.Info("sms (console): no real SMS sent", "phone", phone, "message", message)
	return nil
}

// KavenegarSender sends SMS through Kavenegar's REST API.
type KavenegarSender struct {
	APIKey     string
	Sender     string
	HTTPClient *http.Client
}

// kavenegarResponse is the subset of Kavenegar's send response we inspect.
type kavenegarResponse struct {
	Return struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"return"`
}

// Send posts the message to Kavenegar's send endpoint.
func (s *KavenegarSender) Send(phone, message string) error {
	if s.APIKey == "" {
		return fmt.Errorf("kavenegar api_key is empty")
	}
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	form := url.Values{}
	form.Set("receptor", phone)
	form.Set("message", message)
	if s.Sender != "" {
		form.Set("sender", s.Sender)
	}

	endpoint := fmt.Sprintf("https://api.kavenegar.com/v1/%s/sms/send.json", s.APIKey)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build kavenegar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("kavenegar send: %w", err)
	}
	defer resp.Body.Close()

	var parsed kavenegarResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("decode kavenegar response: %w", err)
	}
	if parsed.Return.Status != http.StatusOK {
		return fmt.Errorf("kavenegar send failed: %s", parsed.Return.Message)
	}
	return nil
}

// New selects a sender by provider name ("console" or "kavenegar"); any
// unrecognized provider falls back to the console sender (dev-safe default).
func New(provider, apiKey, sender string, log *slog.Logger) SmsSender {
	if provider == "kavenegar" {
		return &KavenegarSender{APIKey: apiKey, Sender: sender}
	}
	return &ConsoleSender{Log: log}
}
