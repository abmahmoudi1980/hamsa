package payment

import (
	"net/url"
	"strings"
	"testing"
)

// The browser hand-off (003-web-frontend F3) is a redirect built from a
// client-supplied path, so it is an open-redirect risk by construction. These
// tests pin both halves: the happy path produces a usable same-origin target,
// and every traversal shape is refused.

func TestCallbackURLForAttachesReturnPath(t *testing.T) {
	s := &PaymentService{callback: "https://api.example/api/v1/payments/callback", webReturn: "https://app.example"}

	got := s.callbackURLFor("/payment/result")

	if got != "https://api.example/api/v1/payments/callback?return_path=%2Fpayment%2Fresult" {
		t.Fatalf("callbackURLFor = %q", got)
	}
}

func TestCallbackURLForOmitsUnsafePaths(t *testing.T) {
	s := &PaymentService{callback: "https://api.example/cb", webReturn: "https://app.example"}

	// Protocol-relative, absolute, and backslash forms must all fall back to the
	// bare callback so the gateway never carries an attacker-chosen origin.
	for _, path := range []string{
		"//evil.example/steal",
		"https://evil.example/steal",
		"/\\evil.example",
		"payment/result",
	} {
		if got := s.callbackURLFor(path); got != "https://api.example/cb" {
			t.Errorf("callbackURLFor(%q) = %q, want the bare callback", path, got)
		}
	}
}

func TestCallbackURLForIgnoresPathWithoutWebReturn(t *testing.T) {
	// Android-only deployment: no web_return_url configured, so the pay URL
	// must stay byte-identical to the legacy callback.
	s := &PaymentService{callback: "https://api.example/cb"}

	if got := s.callbackURLFor("/payment/result"); got != "https://api.example/cb" {
		t.Fatalf("callbackURLFor = %q, want bare callback when web_return is unset", got)
	}
}

func TestCallbackURLForAppendsToExistingQuery(t *testing.T) {
	s := &PaymentService{callback: "https://api.example/cb?v=2", webReturn: "https://app.example"}

	got := s.callbackURLFor("/payment/result")

	if strings.Count(got, "?") != 1 || !strings.Contains(got, "&return_path=") {
		t.Fatalf("callbackURLFor = %q, want an & separator", got)
	}
}

func TestBrowserReturnURLBuildsSameOriginTarget(t *testing.T) {
	s := &PaymentService{webReturn: "https://app.example/"}

	got := s.BrowserReturnURL("/payment/result", "pay-1", "verified", "inv-9")

	base, err := url.Parse(got)
	if err != nil {
		t.Fatalf("result is not a URL: %v", err)
	}
	if base.Scheme != "https" || base.Host != "app.example" {
		t.Fatalf("redirect origin = %s://%s, want https://app.example", base.Scheme, base.Host)
	}
	if base.Path != "/payment/result" {
		t.Fatalf("path = %q", base.Path)
	}
	q := base.Query()
	if q.Get("payment_id") != "pay-1" || q.Get("status") != "verified" || q.Get("invoice_id") != "inv-9" {
		t.Fatalf("query = %v", q)
	}
}

func TestBrowserReturnURLRefusesForeignOrigins(t *testing.T) {
	s := &PaymentService{webReturn: "https://app.example"}

	// A tampered return_path on the callback must land on the configured
	// origin's default route, never on the attacker's host.
	for _, path := range []string{
		"https://evil.example/steal",
		"//evil.example",
		"/\\evil.example",
	} {
		got := s.BrowserReturnURL(path, "pay-1", "verified", "")
		if !strings.HasPrefix(got, "https://app.example/payment/result?") {
			t.Errorf("BrowserReturnURL(%q) = %q, want the default same-origin route", path, got)
		}
	}
}

func TestBrowserReturnURLEmptyWhenUnconfigured(t *testing.T) {
	// Android-only: the callback must keep returning JSON + deeplink, so the
	// handler must see "" and skip the redirect entirely.
	s := &PaymentService{}

	if got := s.BrowserReturnURL("/payment/result", "pay-1", "verified", ""); got != "" {
		t.Fatalf("BrowserReturnURL = %q, want empty when web_return is unset", got)
	}
}

func TestBrowserReturnURLRoundTripsThroughCallbackURL(t *testing.T) {
	// End-to-end of the wire contract: the path the client sent to
	// POST /invoices/{id}/pay comes back on the callback query, and produces
	// the redirect target. This is the exact sequence the browser follows.
	s := &PaymentService{callback: "https://api.example/cb", webReturn: "https://app.example"}

	callbackURL := s.callbackURLFor("/payment/result")

	u, err := url.Parse(callbackURL)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := s.BrowserReturnURL(u.Query().Get("return_path"), "pay-7", "verified", "inv-3")

	want := "https://app.example/payment/result?invoice_id=inv-3&payment_id=pay-7&status=verified"
	if got != want {
		t.Fatalf("redirect = %q, want %q", got, want)
	}
}
