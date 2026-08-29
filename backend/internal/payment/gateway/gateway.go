// Package gateway defines the provider-agnostic PaymentGateway contract
// (research.md R5) and its adapters: Zarinpal (v4 JSON API) for production
// and a deterministic mock for dev/tests.
package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StartRequest is the payload for initiating a gateway payment. Amounts are
// Toman — adapters convert to the provider's unit internally (Rial for
// Zarinpal).
type StartRequest struct {
	AmountToman int64
	CallbackURL string
	Description string
}

// StartResult carries the gateway redirect URL and the provider's opaque
// authority token (stored on the payment row; the callback resolves by it).
type StartResult struct {
	PayURL    string
	Authority string
}

// VerifyResult is the outcome of a verification call.
type VerifyResult struct {
	OK    bool
	RefID string // provider tracking/reference number on success
}

// ErrVerifyFailed means the gateway reported the transaction as not
// successful (cancelled, expired, or rejected).
var ErrVerifyFailed = errors.New("gateway: verification failed")

// ErrAmountMismatch means the gateway rejected the verification because the
// amount sent does not match the started transaction (tamper protection —
// verification amounts always come from our DB, never the callback).
var ErrAmountMismatch = errors.New("gateway: amount mismatch")

// PaymentGateway abstracts the online payment provider.
type PaymentGateway interface {
	// Name returns the provider key stored on the payment row ("zarinpal", "mock").
	Name() string
	// Start initiates a payment and returns the redirect URL + authority.
	Start(ctx context.Context, req StartRequest) (StartResult, error)
	// Verify confirms a started transaction. amountToman MUST be the amount
	// recorded in our database when the payment was started — never anything
	// the callback carried (research.md R5).
	Verify(ctx context.Context, authority string, amountToman int64) (VerifyResult, error)
}

// --- Zarinpal (v4) -----------------------------------------------------------------

const (
	zarinpalRequestURLFmt = "https://%s/pg/v4/payment/request.json"
	zarinpalVerifyURLFmt  = "https://%s/pg/v4/payment/verify.json"
	zarinpalStartPayFmt   = "https://%s/pg/StartPay/%s"
)

// Zarinpal talks to the Zarinpal v4 JSON API (sandbox or production host).
type Zarinpal struct {
	MerchantID string
	Sandbox    bool
	HTTPClient *http.Client
}

// NewZarinpal returns a Zarinpal adapter.
func NewZarinpal(merchantID string, sandbox bool) *Zarinpal {
	return &Zarinpal{
		MerchantID: merchantID,
		Sandbox:    sandbox,
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
	}
}

// Name implements PaymentGateway.
func (z *Zarinpal) Name() string { return "zarinpal" }

func (z *Zarinpal) host() string {
	if z.Sandbox {
		return "sandbox.zarinpal.com"
	}
	return "payment.zarinpal.com"
}

// zarinpalRial converts Toman to Rial (×10) — done ONLY inside the adapter
// (research.md R5).
func zarinpalRial(toman int64) int64 { return toman * 10 }

type zarinpalEnvelope struct {
	Data   map[string]any `json:"data"`
	Errors map[string]any `json:"errors"`
}

// Start implements PaymentGateway.
func (z *Zarinpal) Start(ctx context.Context, req StartRequest) (StartResult, error) {
	payload := map[string]any{
		"merchant_id":  z.MerchantID,
		"amount":       zarinpalRial(req.AmountToman),
		"callback_url": req.CallbackURL,
		"description":  req.Description,
	}
	var env zarinpalEnvelope
	if err := z.post(ctx, fmt.Sprintf(zarinpalRequestURLFmt, z.host()), payload, &env); err != nil {
		return StartResult{}, err
	}
	if len(env.Errors) > 0 || env.Data == nil {
		return StartResult{}, fmt.Errorf("zarinpal: request rejected: %v", env.Errors)
	}
	authority, _ := env.Data["authority"].(string)
	if authority == "" {
		return StartResult{}, fmt.Errorf("zarinpal: missing authority: %v", env.Data)
	}
	return StartResult{
		PayURL:    fmt.Sprintf(zarinpalStartPayFmt, z.host(), authority),
		Authority: authority,
	}, nil
}

// Verify implements PaymentGateway. Code 100 = verified; 101 = already
// verified (idempotent success).
func (z *Zarinpal) Verify(ctx context.Context, authority string, amountToman int64) (VerifyResult, error) {
	payload := map[string]any{
		"merchant_id": z.MerchantID,
		"amount":      zarinpalRial(amountToman), // from OUR DB — never the callback
		"authority":   authority,
	}
	var env zarinpalEnvelope
	if err := z.post(ctx, fmt.Sprintf(zarinpalVerifyURLFmt, z.host()), payload, &env); err != nil {
		return VerifyResult{}, err
	}
	if len(env.Errors) > 0 {
		return VerifyResult{}, fmt.Errorf("%w: %v", ErrVerifyFailed, env.Errors)
	}
	code, _ := env.Data["code"].(float64)
	switch int64(code) {
	case 100, 101:
		ref, _ := env.Data["ref_id"].(string)
		return VerifyResult{OK: true, RefID: ref}, nil
	default:
		return VerifyResult{}, fmt.Errorf("%w: code %v", ErrVerifyFailed, code)
	}
}

func (z *Zarinpal) post(ctx context.Context, url string, payload any, out *zarinpalEnvelope) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, jsonBody(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := z.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("zarinpal: %w", err)
	}
	defer resp.Body.Close()
	return decodeJSON(resp.Body, out)
}

// --- Mock (dev/tests) ---------------------------------------------------------------

// Mock is a deterministic gateway for dev and integration tests. FailNext /
// MismatchNext force the next Verify call to fail, exercising the failure
// and amount-mismatch paths without a real provider.
type Mock struct {
	FailNext     bool
	MismatchNext bool
}

// NewMock returns the mock gateway.
func NewMock() *Mock { return &Mock{} }

// Name implements PaymentGateway.
func (m *Mock) Name() string { return "mock" }

// Start implements PaymentGateway. The redirect URL points back at our own
// callback endpoint with the authority attached — in dev the "gateway page"
// is a no-op the client can open directly.
func (m *Mock) Start(_ context.Context, req StartRequest) (StartResult, error) {
	authority := "MOCK-" + time.Now().UTC().Format("20060102150405.000000000")
	sep := "?"
	if strings.Contains(req.CallbackURL, "?") {
		sep = "&"
	}
	return StartResult{
		PayURL:    req.CallbackURL + sep + "Authority=" + authority,
		Authority: authority,
	}, nil
}

// Verify implements PaymentGateway.
func (m *Mock) Verify(_ context.Context, _ string, _ int64) (VerifyResult, error) {
	switch {
	case m.MismatchNext:
		m.MismatchNext = false
		return VerifyResult{}, ErrAmountMismatch
	case m.FailNext:
		m.FailNext = false
		return VerifyResult{}, ErrVerifyFailed
	default:
		return VerifyResult{OK: true, RefID: "MOCKREF-" + time.Now().UTC().Format("150405")}, nil
	}
}

// jsonBody marshals payload into a reader (small helper shared by adapters).
func jsonBody(payload any) io.Reader {
	b, _ := json.Marshal(payload)
	return bytes.NewReader(b)
}

// decodeJSON decodes a JSON response body into out.
func decodeJSON(r io.Reader, out any) error {
	return json.NewDecoder(r).Decode(out)
}
