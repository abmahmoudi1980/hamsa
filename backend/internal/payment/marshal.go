package payment

// Wire marshaling: money values serialize as quoted integer strings on the
// wire (contracts/api.md — avoids client precision loss); columns stay
// integer Toman.

import (
	"encoding/json"
	"strconv"
)

// MarshalJSON renders every money field as a quoted integer string.
func (b UnitBalance) MarshalJSON() ([]byte, error) {
	type alias UnitBalance
	return json.Marshal(struct {
		alias
		PriorDebt            string `json:"prior_debt"`
		CurrentInvoiceAmount string `json:"current_invoice_amount"`
		LateFeeTotal         string `json:"late_fee_total"`
		Credit               string `json:"credit"`
		PaidTotal            string `json:"paid_total"`
		CreditAsset          string `json:"credit_asset"`
		Balance              string `json:"balance"`
	}{
		alias:                (alias)(b),
		PriorDebt:            strconv.FormatInt(b.PriorDebt, 10),
		CurrentInvoiceAmount: strconv.FormatInt(b.CurrentInvoiceAmount, 10),
		LateFeeTotal:         strconv.FormatInt(b.LateFeeTotal, 10),
		Credit:               strconv.FormatInt(b.Credit, 10),
		PaidTotal:            strconv.FormatInt(b.PaidTotal, 10),
		CreditAsset:          strconv.FormatInt(b.CreditAsset, 10),
		Balance:              strconv.FormatInt(b.Balance, 10),
	})
}

// MarshalJSON renders the amount as a quoted integer string.
func (p Payment) MarshalJSON() ([]byte, error) {
	type alias Payment
	return json.Marshal(struct {
		alias
		Amount string `json:"amount"`
	}{
		alias:  (alias)(p),
		Amount: strconv.FormatInt(p.Amount, 10),
	})
}
