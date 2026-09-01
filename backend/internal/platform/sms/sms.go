// Package sms provides an SMS sending abstraction for OTP delivery: a
// console/log sender for development and HTTP adapters for Kavenegar and
// SMS.ir in production (research.md R4).
//
// Senders receive the raw OTP code and compose the message themselves:
// Kavenegar builds the free text, while SMS.ir substitutes the code into a
// template defined in the SMS.ir panel via the verify endpoint.
package sms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SmsSender delivers an OTP code to a single recipient phone number.
type SmsSender interface {
	SendCode(phone, code string) error
}

// Options selects and configures the provider ("console", "kavenegar", or
// "smsir").
type Options struct {
	Provider  string
	Kavenegar KavenegarSender
	SmsIR     SmsIRSender
	Log       *slog.Logger
}

// New selects a sender by provider name; any unrecognized provider falls back
// to the console sender (dev-safe default).
func New(opts Options) SmsSender {
	switch opts.Provider {
	case "kavenegar":
		return &opts.Kavenegar
	case "smsir":
		return &opts.SmsIR
	default:
		return &ConsoleSender{Log: opts.Log}
	}
}

// otpMessage is the SMS body used by free-text providers.
func otpMessage(code string) string {
	return fmt.Sprintf("کد ورود شما به همسا: %s", code)
}

// ConsoleSender logs the code instead of sending it — development only.
type ConsoleSender struct {
	Log *slog.Logger
}

// SendCode logs the code; no real SMS is dispatched.
func (s *ConsoleSender) SendCode(phone, code string) error {
	log := s.Log
	if log == nil {
		log = slog.Default()
	}
	log.Info("sms (console): no real SMS sent", "phone", phone, "code", code)
	return nil
}

// KavenegarSender sends OTP codes as free text through Kavenegar's REST API.
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

// SendCode posts the OTP message to Kavenegar's send endpoint.
func (s *KavenegarSender) SendCode(phone, code string) error {
	if s.APIKey == "" {
		return fmt.Errorf("kavenegar api_key is empty")
	}
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	form := url.Values{}
	form.Set("receptor", phone)
	form.Set("message", otpMessage(code))
	if s.Sender != "" {
		form.Set("sender", s.Sender)
	}

	endpoint := fmt.Sprintf("https://api.kavenegar.com/v1/%s/sms/send.json", s.APIKey)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))

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

// SmsIRSender sends OTP codes through SMS.ir's verify endpoint
// (https://api.sms.ir/v1/send/verify). The code is substituted into a
// template created in the SMS.ir panel; the API supplies only parameter
// values, so TemplateID and ParamName must match the panel template
type SmsIRSender struct {
	APIKey     string
	TemplateID int64
	ParamName  string // template parameter name, without the surrounding '#'
	BaseURL    string // production default https://api.sms.ir
	HTTPClient *http.Client
}

// smsirResponse is the uniform SMS.ir envelope; status 1 means success.
type smsirResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// verifyRequest is the /v1/send/verify body. The parameter value is capped at
// 25 characters (SMS.ir status 114); a 6-digit code always fits.
type verifyRequest struct {
	Mobile     string           `json:"mobile"`
	TemplateID int64            `json:"templateId"`
	Parameters []smsirParameter `json:"parameters"`
}

type smsirParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SendCode posts the code to SMS.ir's verify endpoint.
func (s *SmsIRSender) SendCode(phone, code string) error {
	if s.APIKey == "" {
		return fmt.Errorf("sms.ir api_key is empty")
	}
	if s.TemplateID == 0 {
		return fmt.Errorf("sms.ir template_id is empty")
	}
	name := s.ParamName
	if name == "" {
		name = "Code"
	}
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	body, err := json.Marshal(verifyRequest{
		Mobile:     NormalizeIranianMobile(phone),
		TemplateID: s.TemplateID,
		Parameters: []smsirParameter{{Name: name, Value: code}},
	})
	if err != nil {
		return fmt.Errorf("build sms.ir request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, s.baseURL()+"/v1/send/verify", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build sms.ir request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-KEY", s.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sms.ir verify: %w", err)
	}
	defer resp.Body.Close()

	var parsed smsirResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("decode sms.ir response: %w", err)
	}
	if parsed.Status != 1 {
		return fmt.Errorf("sms.ir verify failed (status %d): %s", parsed.Status, parsed.Message)
	}
	return nil
}

// baseURL returns the configured API host or the production default.
func (s *SmsIRSender) baseURL() string {
	if s.BaseURL != "" {
		return strings.TrimSuffix(s.BaseURL, "/")
	}
	return "https://api.sms.ir"
}

// NormalizeIranianMobile converts 09XXXXXXXXX, +989XXXXXXXXX, and
// 989XXXXXXXXX to the 9XXXXXXXXX form used in SMS.ir's API samples.
func NormalizeIranianMobile(phone string) string {
	p := strings.TrimSpace(phone)
	if strings.HasPrefix(p, "+98") {
		p = strings.TrimPrefix(p, "+98")
	} else if strings.HasPrefix(p, "98") && len(p) > 10 {
		p = strings.TrimPrefix(p, "98")
	}
	return strings.TrimPrefix(p, "0")
}
