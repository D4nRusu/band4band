package services

import (
	"log"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

const receiptDebugLimit = 64 << 10

func (extractor *ReceiptExtractor) logReceiptResponse(status int, body []byte) {
	if !extractor.Debug {
		return
	}

	preview := body
	if len(preview) > receiptDebugLimit {
		preview = preview[:receiptDebugLimit]
	}

	log.Printf("[receipt-llm] HTTP %d response (showing %d of %d bytes): %s", status, len(preview), len(body), preview)
}

func (extractor *ReceiptExtractor) logReceiptResult(draft models.ReceiptDraft, err error) {
	if !extractor.Debug {
		return
	}

	if err != nil {
		log.Printf("[receipt-llm] rejected response: %v", err)
		return
	}

	log.Printf("[receipt-llm] accepted %d items; totalMinor=%d; itemsTotalMinor=%d; totalsMatch=%t",
		len(draft.Items), draft.TotalMinor, draft.ItemsTotalMinor, draft.TotalsMatch)
}
