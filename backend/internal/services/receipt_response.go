package services

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

var receiptDecimal = regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]{1,2})?$`)
var receiptCurrency = regexp.MustCompile(`^[A-Z]{3}$`)

func decodeReceipt(content string) (models.ReceiptDraft, error) {
	var output struct {
		Currency *string `json:"currency"`
		Total    *string `json:"total"`
		Items    []struct {
			Name   string  `json:"name"`
			Amount *string `json:"amount"`
		} `json:"items"`
	}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return models.ReceiptDraft{}, models.ErrReceiptResponse
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		return models.ReceiptDraft{}, models.ErrReceiptResponse
	}

	if output.Currency == nil || output.Total == nil || len(output.Items) == 0 {
		return models.ReceiptDraft{}, models.ErrReceiptUnreadable
	}

	if !receiptCurrency.MatchString(*output.Currency) || len(output.Items) > 200 {
		return models.ReceiptDraft{}, models.ErrReceiptResponse
	}

	total, err := parseReceiptAmount(*output.Total)
	if err != nil || total <= 0 {
		return models.ReceiptDraft{}, models.ErrReceiptResponse
	}

	draft := models.ReceiptDraft{Currency: *output.Currency, TotalMinor: total, Items: make([]models.ReceiptItem, 0, len(output.Items))}
	for i, item := range output.Items {
		name := strings.TrimSpace(item.Name)
		if name == "" || utf8.RuneCountInString(name) > 200 {
			return models.ReceiptDraft{}, models.ErrReceiptResponse
		}

		if item.Amount == nil {
			return models.ReceiptDraft{}, models.ErrReceiptUnreadable
		}

		amount, err := parseReceiptAmount(*item.Amount)
		if err != nil {
			return models.ReceiptDraft{}, err
		}

		sum := draft.ItemsTotalMinor + amount
		if sum < -models.MaxAmountMinor || sum > models.MaxAmountMinor {
			return models.ReceiptDraft{}, models.ErrReceiptResponse
		}

		draft.ItemsTotalMinor = sum
		draft.Items = append(draft.Items, models.ReceiptItem{ID: fmt.Sprintf("item-%d", i+1), Name: name, AmountMinor: amount})
	}

	draft.TotalsMatch = draft.ItemsTotalMinor == draft.TotalMinor
	return draft, nil
}

func parseReceiptAmount(value string) (int64, error) {
	if len(value) > 32 || !receiptDecimal.MatchString(value) {
		return 0, models.ErrReceiptResponse
	}

	negative := strings.HasPrefix(value, "-")
	whole, fraction, _ := strings.Cut(strings.TrimPrefix(value, "-"), ".")
	digits := whole + fraction + strings.Repeat("0", 2-len(fraction))
	amount, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || amount > models.MaxAmountMinor {
		return 0, models.ErrReceiptResponse
	}

	if negative {
		amount = -amount
	}

	return amount, nil
}
