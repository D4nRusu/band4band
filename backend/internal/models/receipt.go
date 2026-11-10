package models

import "errors"

var (
	ErrReceiptUnavailable = errors.New("receipt extraction is not available")
	ErrReceiptBusy        = errors.New("another receipt is being processed; please retry shortly")
	ErrReceiptUnreadable  = errors.New("could not read the receipt's currency, total, and items; try a clearer photo")
	ErrReceiptResponse    = errors.New("the receipt model returned an invalid or incomplete response")
)

// ReceiptDraft is unconfirmed extraction data, not an expense or financial record.
type ReceiptDraft struct {
	Currency        string        `json:"currency"`
	TotalMinor      int64         `json:"totalMinor"`
	ItemsTotalMinor int64         `json:"itemsTotalMinor"`
	TotalsMatch     bool          `json:"totalsMatch"`
	Items           []ReceiptItem `json:"items"`
}

type ReceiptItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AmountMinor int64  `json:"amountMinor"`
}
