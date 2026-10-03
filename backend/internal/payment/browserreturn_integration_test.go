package payment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestMePaymentsUsesStandardEnvelope is the regression test for defect F1 in
// 003-web-frontend/plan.md: GET /me/payments answered {payments,total} while
// contracts/api.md documents {items,page,page_size,total}. The shipped Flutter
// client read `items` (payment_repository.dart myPayments), so every resident's
// payment history rendered empty on Android. These assertions pin the contract
// shape both the web client and the fixed mobile client depend on.
func TestMePaymentsUsesStandardEnvelope(t *testing.T) {
	e := newPaymentEnv(t)
	mgrID, _ := e.seedUser(t, "manager", "09120000031")
	bID, unitID := e.seedBuildingUnit(t, mgrID)
	resTok := e.residentToken(t, "09120000032")
	e.grantResidency(t, bID, unitID, "09120000032")

	mgrTok := e.login(t, "09120000031", payTestPassword)
	invID := e.issueInvoice(t, mgrTok, bID, unitID, 1_000_000).String()
	if code, _ := e.post(t, "/api/v1/invoices/"+invID+"/payments", mgrTok,
		map[string]any{"amount": "400000", "paid_at": "2026-08-01"}); code != http.StatusCreated {
		t.Fatalf("record payment: %d", code)
	}

	code, body := e.get(t, "/api/v1/me/payments", resTok)
	if code != http.StatusOK {
		t.Fatalf("me/payments: %d", code)
	}

	for _, key := range []string{"items", "page", "page_size", "total"} {
		if _, ok := body[key]; !ok {
			t.Errorf("envelope is missing %q; got keys %v", key, keysOf(body))
		}
	}
	items, ok := body["items"].([]any)
	if !ok {
		t.Fatalf("items is %T, want an array — this is exactly the F1 breakage", body["items"])
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1 (the payment this resident just made)", len(items))
	}
	if got := num(body, "total"); got != 1 {
		t.Fatalf("total = %d, want 1", got)
	}
}

// TestBuildingLedgerUsesStandardEnvelope pins the same contract on the manager
// ledger, which previously used ?size= and returned only {payments,total} with
// no page echo.
func TestBuildingLedgerUsesStandardEnvelope(t *testing.T) {
	e := newPaymentEnv(t)
	mgrID, _ := e.seedUser(t, "manager", "09120000041")
	bID, unitID := e.seedBuildingUnit(t, mgrID)
	mgrTok := e.login(t, "09120000041", payTestPassword)

	invID := e.issueInvoice(t, mgrTok, bID, unitID, 1_000_000).String()
	if code, _ := e.post(t, "/api/v1/invoices/"+invID+"/payments", mgrTok,
		map[string]any{"amount": "250000", "paid_at": "2026-08-01"}); code != http.StatusCreated {
		t.Fatalf("record payment: %d", code)
	}

	code, body := e.get(t, "/api/v1/buildings/"+bID.String()+"/payments?page=1&page_size=10", mgrTok)
	if code != http.StatusOK {
		t.Fatalf("ledger: %d", code)
	}
	if got := num(body, "page"); got != 1 {
		t.Errorf("page = %d, want 1", got)
	}
	if got := num(body, "page_size"); got != 10 {
		t.Errorf("page_size = %d, want 10 (the standard param must be honored)", got)
	}
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %v, want 1 payment", body["items"])
	}
}

// TestPaymentListAcceptsLegacySizeParam keeps already-installed APKs working:
// the pre-003 client sends ?size=, and it must not be silently ignored.
func TestPaymentListAcceptsLegacySizeParam(t *testing.T) {
	e := newPaymentEnv(t)
	mgrID, _ := e.seedUser(t, "manager", "09120000051")
	bID, unitID := e.seedBuildingUnit(t, mgrID)
	mgrTok := e.login(t, "09120000051", payTestPassword)

	invID := e.issueInvoice(t, mgrTok, bID, unitID, 1_000_000).String()
	if code, _ := e.post(t, "/api/v1/invoices/"+invID+"/payments", mgrTok,
		map[string]any{"amount": "100000", "paid_at": "2026-08-01"}); code != http.StatusCreated {
		t.Fatalf("record payment: %d", code)
	}

	code, body := e.get(t, "/api/v1/buildings/"+bID.String()+"/payments?page=1&size=7", mgrTok)
	if code != http.StatusOK {
		t.Fatalf("ledger: %d", code)
	}
	if got := num(body, "page_size"); got != 7 {
		t.Fatalf("page_size = %d, want 7 — the legacy ?size= alias must still be honored", got)
	}
}

// TestPaymentListKeepsDeprecatedPaymentsAlias guards the shipped APK during the
// one release cycle the alias exists for.
func TestPaymentListKeepsDeprecatedPaymentsAlias(t *testing.T) {
	e := newPaymentEnv(t)
	mgrID, _ := e.seedUser(t, "manager", "09120000061")
	bID, _ := e.seedBuildingUnit(t, mgrID)
	mgrTok := e.login(t, "09120000061", payTestPassword)

	_, body := e.get(t, "/api/v1/buildings/"+bID.String()+"/payments", mgrTok)
	if _, ok := body["payments"]; !ok {
		t.Fatal("deprecated `payments` alias is missing; a pre-003 APK would render an empty ledger")
	}
}

// TestGatewayCallbackRedirectsBrowserToReturnPath is the F3 browser hand-off:
// a web client that sent return_path on POST /invoices/{id}/pay must land back
// on its own result route, not on a JSON blob.
func TestGatewayCallbackRedirectsBrowserToReturnPath(t *testing.T) {
	e := newPaymentEnvWithWebReturn(t, "https://app.example")
	mgrID, _ := e.seedUser(t, "manager", "09120000071")
	bID, unitID := e.seedBuildingUnit(t, mgrID)
	e.grantResidency(t, bID, unitID, "09120000072")
	resTok := e.residentToken(t, "09120000072")

	mgrTok := e.login(t, "09120000071", payTestPassword)
	invID := e.issueInvoice(t, mgrTok, bID, unitID, 1_000_000).String()

	code, body := e.post(t, "/api/v1/invoices/"+invID+"/pay", resTok,
		map[string]any{"return_path": "/payment/result"})
	if code != http.StatusCreated {
		t.Fatalf("start pay: %d %v", code, body)
	}
	payURL, _ := body["payment_url"].(string)
	if !strings.Contains(payURL, "return_path=%2Fpayment%2Fresult") {
		t.Fatalf("payment_url %q does not carry the return path", payURL)
	}

	// Follow the gateway redirect the way a browser would. The mock gateway's
	// payment_url is built from the configured callback host, which is not the
	// host the test engine serves, so only its path+query are replayed.
	authority := e.authorityFor(t, body["payment_id"].(string))
	code, loc := e.callbackRedirect(t, "/api/v1/payments/callback?Authority="+authority)
	if code != http.StatusFound {
		t.Fatalf("callback status = %d, want 302 to the browser result route", code)
	}
	if !strings.HasPrefix(loc, "https://app.example/payment/result?") {
		t.Fatalf("Location = %q, want the configured web origin", loc)
	}
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	q := u.Query()
	if q.Get("payment_id") == "" || q.Get("status") == "" {
		t.Fatalf("Location query %v must carry payment_id and status", q)
	}
	if q.Get("invoice_id") != invID {
		t.Errorf("invoice_id = %q, want %q so the client can link to the invoice", q.Get("invoice_id"), invID)
	}
}

// TestGatewayCallbackIgnoresForeignReturnPath is the open-redirect guard at the
// HTTP layer: a tampered return_path on the callback URL must not escape the
// configured origin.
func TestGatewayCallbackIgnoresForeignReturnPath(t *testing.T) {
	e := newPaymentEnvWithWebReturn(t, "https://app.example")
	mgrID, _ := e.seedUser(t, "manager", "09120000081")
	bID, unitID := e.seedBuildingUnit(t, mgrID)
	e.grantResidency(t, bID, unitID, "09120000082")
	resTok := e.residentToken(t, "09120000082")

	mgrTok := e.login(t, "09120000081", payTestPassword)
	invID := e.issueInvoice(t, mgrTok, bID, unitID, 1_000_000).String()

	_, body := e.post(t, "/api/v1/invoices/"+invID+"/pay", resTok, nil)
	authority := e.authorityFor(t, body["payment_id"].(string))

	// The attacker swaps in their own host on the callback URL.
	code, loc := e.callbackRedirect(t, "/api/v1/payments/callback?Authority="+authority+
		"&return_path=https%3A%2F%2Fevil.example%2Fsteal")
	if code != http.StatusFound {
		t.Fatalf("callback status = %d, want 302", code)
	}
	if strings.Contains(loc, "evil.example") {
		t.Fatalf("Location = %q — open redirect: the callback followed a foreign origin", loc)
	}
	if !strings.HasPrefix(loc, "https://app.example/payment/result") {
		t.Fatalf("Location = %q, want the default same-origin route", loc)
	}
}

// TestGatewayCallbackWithoutWebReturnStaysJSON pins the Android-only posture:
// with no web_return_url configured the callback must keep its JSON + deeplink
// response so the shipped APK flow is untouched.
func TestGatewayCallbackWithoutWebReturnStaysJSON(t *testing.T) {
	e := newPaymentEnv(t) // webReturn = ""
	mgrID, _ := e.seedUser(t, "manager", "09120000091")
	bID, unitID := e.seedBuildingUnit(t, mgrID)
	e.grantResidency(t, bID, unitID, "09120000092")
	resTok := e.residentToken(t, "09120000092")

	mgrTok := e.login(t, "09120000091", payTestPassword)
	invID := e.issueInvoice(t, mgrTok, bID, unitID, 1_000_000).String()

	_, body := e.post(t, "/api/v1/invoices/"+invID+"/pay", resTok, nil)
	authority := e.authorityFor(t, body["payment_id"].(string))

	code, out := e.get(t, "/api/v1/payments/callback?Authority="+authority, "")
	if code != http.StatusOK {
		t.Fatalf("callback: %d", code)
	}
	if _, ok := out["deeplink"]; !ok {
		t.Fatal("deeplink missing — the Android hand-off regressed")
	}
}

// newPaymentEnvWithWebReturn builds the standard payment env with a browser
// return origin configured (app.web_return_url), for the 003 hand-off tests.
func newPaymentEnvWithWebReturn(t *testing.T, webReturn string) *payEnv {
	t.Helper()
	return newPaymentEnvWebReturn(t, webReturn)
}

// residentToken seeds a resident with a known password and returns their access
// token, obtained through the real login route.
func (e *payEnv) residentToken(t *testing.T, phone string) string {
	t.Helper()
	_, _ = e.seedUserWithPassword(t, "resident", phone, payTestPassword)
	return e.login(t, phone, payTestPassword)
}

// authorityFor reads the gateway authority for a payment, standing in for what
// the gateway appends to the redirect URL.
func (e *payEnv) authorityFor(t *testing.T, paymentID string) string {
	t.Helper()
	var authority string
	if err := e.gormDB.WithContext(context.Background()).
		Raw(`SELECT authority FROM payments WHERE id = ?`, uuid.MustParse(paymentID)).
		Scan(&authority).Error; err != nil || authority == "" {
		t.Fatalf("lookup authority: %v %q", err, authority)
	}
	return authority
}

// callbackRedirect issues the gateway callback request without following
// redirects and returns the status + Location header. An absolute URL is
// reduced to its path+query, since the test engine serves on a different host
// than the callback URL the mock gateway was handed.
func (e *payEnv) callbackRedirect(t *testing.T, target string) (int, string) {
	t.Helper()
	path := target
	if u, err := url.Parse(target); err == nil && u.Scheme != "" {
		path = u.RequestURI()
	}
	w := httptest.NewRecorder()
	e.engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Code, w.Header().Get("Location")
}

// keysOf lists the top-level envelope keys, for readable failure messages.
func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
