package sms

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestNormalizeIranianMobile(t *testing.T) {
	cases := map[string]string{
		"09121234567":     "9121234567",
		" +989121234567 ": "9121234567",
		"989121234567":    "9121234567",
		"9121234567":      "9121234567",
	}
	for in, want := range cases {
		if got := NormalizeIranianMobile(in); got != want {
			t.Errorf("NormalizeIranianMobile(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSmsIRSender_Success(t *testing.T) {
	var gotPath, gotMethod string
	var gotHeaders http.Header
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod, gotHeaders = r.URL.Path, r.Method, r.Header
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		io.WriteString(w, `{"status":1,"message":"موفق","data":{"messageId":89545112,"cost":1.0}}`)
	}))
	defer srv.Close()

	s := &SmsIRSender{APIKey: "test-key", TemplateID: 123456, BaseURL: srv.URL}
	if err := s.SendCode("09121234567", "123456"); err != nil {
		t.Fatalf("SendCode: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/send/verify" {
		t.Fatalf("expected POST /v1/send/verify, got %s %s", gotMethod, gotPath)
	}
	if gotHeaders.Get("X-API-KEY") != "test-key" {
		t.Fatalf("missing X-API-KEY header: %v", gotHeaders)
	}
	if gotBody["mobile"] != "9121234567" {
		t.Fatalf("mobile not normalized: %v", gotBody["mobile"])
	}
	if gotBody["templateId"].(float64) != 123456 {
		t.Fatalf("templateId: %v", gotBody["templateId"])
	}
	params, ok := gotBody["parameters"].([]any)
	if !ok || len(params) != 1 {
		t.Fatalf("parameters: %v", gotBody["parameters"])
	}
	p := params[0].(map[string]any)
	if p["name"] != "Code" || p["value"] != "123456" {
		t.Fatalf("parameter: %v", p)
	}
}

func TestSmsIRSender_DefaultParamName(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		json.Unmarshal(raw, &gotBody)
		io.WriteString(w, `{"status":1,"message":"موفق","data":{"messageId":1,"cost":1.0}}`)
	}))
	defer srv.Close()

	s := &SmsIRSender{APIKey: "k", TemplateID: 1, BaseURL: srv.URL}
	if err := s.SendCode("09121234567", "654321"); err != nil {
		t.Fatalf("SendCode: %v", err)
	}
	p := gotBody["parameters"].([]any)[0].(map[string]any)
	if p["name"] != "Code" {
		t.Fatalf("expected default param name Code, got %v", p["name"])
	}
}

func TestSmsIRSender_APIErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int // JSON "status" field
	}{
		{"template-not-found", 113},
		{"invalid-mobile", 104},
		{"service-error", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, `{"status":`+strconv.Itoa(tc.status)+`,"message":"خطا"}`)
			}))
			defer srv.Close()

			s := &SmsIRSender{APIKey: "k", TemplateID: 1, BaseURL: srv.URL}
			err := s.SendCode("09121234567", "123456")
			if err == nil {
				t.Fatalf("expected error for status %d", tc.status)
			}
			if !strings.Contains(err.Error(), "status "+strconv.Itoa(tc.status)) {
				t.Fatalf("error should mention the status, got %q", err.Error())
			}
		})
	}
}

func TestSmsIRSender_MissingConfig(t *testing.T) {
	if err := (&SmsIRSender{}).SendCode("09121234567", "123456"); err == nil {
		t.Fatal("expected error for empty api_key/template_id")
	}
}
