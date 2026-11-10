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
var receiptQuantity = regexp.MustCompile(`^([0-9]+(?:[.,][0-9]+)?)(?:\s*\p{L}+\.?)?$`)

func decodeReceiptCompletion(body []byte) (models.ReceiptDraft, error) {
	var completion struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &completion); err != nil {
		return models.ReceiptDraft{}, fmt.Errorf("%w: completion envelope JSON: %v", models.ErrReceiptResponse, err)
	}

	if len(completion.Choices) != 1 {
		return models.ReceiptDraft{}, fmt.Errorf("%w: expected one completion choice, received %d", models.ErrReceiptResponse, len(completion.Choices))
	}

	choice := completion.Choices[0]
	if choice.FinishReason != "stop" {
		return models.ReceiptDraft{}, fmt.Errorf("%w: finish_reason=%q, expected stop (length means the generation token limit was reached)", models.ErrReceiptResponse, choice.FinishReason)
	}

	return decodeReceipt(choice.Message.Content)
}

func decodeReceipt(content string) (models.ReceiptDraft, error) {
	var output struct {
		Currency *string          `json:"currency"`
		Total    *json.RawMessage `json:"total"`
		Items    []struct {
			Name     string           `json:"name"`
			Quantity json.RawMessage  `json:"quantity"`
			Amount   *json.RawMessage `json:"amount"`
			Discount json.RawMessage  `json:"discount"` // Line amount already includes any line discount.
		} `json:"items"`
	}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return models.ReceiptDraft{}, fmt.Errorf("%w: receipt JSON: %v", models.ErrReceiptResponse, err)
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		return models.ReceiptDraft{}, fmt.Errorf("%w: receipt contains trailing data", models.ErrReceiptResponse)
	}

	if output.Currency == nil || output.Total == nil || len(output.Items) == 0 {
		return models.ReceiptDraft{}, fmt.Errorf("%w: currency or total is missing/null, or items is empty", models.ErrReceiptUnreadable)
	}

	currency := *output.Currency
	if strings.EqualFold(strings.TrimSpace(currency), "lei") {
		currency = "RON"
	}

	if !receiptCurrency.MatchString(currency) {
		return models.ReceiptDraft{}, fmt.Errorf("%w: currency must be a three-letter uppercase code", models.ErrReceiptResponse)
	}

	if len(output.Items) > 200 {
		return models.ReceiptDraft{}, fmt.Errorf("%w: received %d items, limit is 200", models.ErrReceiptResponse, len(output.Items))
	}

	total, err := parseReceiptAmount(*output.Total)
	if err != nil {
		return models.ReceiptDraft{}, fmt.Errorf("receipt total: %w", err)
	}

	if total <= 0 {
		return models.ReceiptDraft{}, fmt.Errorf("%w: receipt total must be positive", models.ErrReceiptResponse)
	}

	draft := models.ReceiptDraft{Currency: currency, TotalMinor: total, Items: make([]models.ReceiptItem, 0, len(output.Items))}
	for i, item := range output.Items {
		name := strings.TrimSpace(item.Name)
		if name == "" || utf8.RuneCountInString(name) > 200 {
			return models.ReceiptDraft{}, fmt.Errorf("%w: item %d name is empty or longer than 200 characters", models.ErrReceiptResponse, i+1)
		}

		if item.Amount == nil {
			return models.ReceiptDraft{}, fmt.Errorf("%w: item %d amount is missing/null", models.ErrReceiptUnreadable, i+1)
		}

		amount, err := parseReceiptAmount(*item.Amount)
		if err != nil {
			return models.ReceiptDraft{}, fmt.Errorf("item %d amount: %w", i+1, err)
		}

		sum := draft.ItemsTotalMinor + amount
		if sum < -models.MaxAmountMinor || sum > models.MaxAmountMinor {
			return models.ReceiptDraft{}, fmt.Errorf("%w: item sum exceeds the supported range at item %d", models.ErrReceiptResponse, i+1)
		}

		draft.ItemsTotalMinor = sum
		draft.Items = append(draft.Items, models.ReceiptItem{
			ID:          fmt.Sprintf("item-%d", i+1),
			Name:        name,
			Quantity:    parseReceiptQuantity(item.Quantity),
			AmountMinor: amount,
		})
	}

	draft.TotalsMatch = draft.ItemsTotalMinor == draft.TotalMinor
	return draft, nil
}

func parseReceiptQuantity(raw json.RawMessage) *float64 {
	value := string(raw)
	if strings.HasPrefix(value, `"`) {
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil
		}
	}

	match := receiptQuantity.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return nil
	}

	quantity, err := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", "."), 64)
	if err != nil || quantity <= 0 {
		return nil
	}

	return &quantity
}

func parseReceiptAmount(raw json.RawMessage) (int64, error) {
	// Keep JSON numbers as decimal text so converting to cents never rounds.
	value := string(raw)
	if strings.HasPrefix(value, `"`) {
		if err := json.Unmarshal(raw, &value); err != nil {
			return 0, fmt.Errorf("%w: invalid quoted amount: %v", models.ErrReceiptResponse, err)
		}
	}

	if len(value) > 32 || !receiptDecimal.MatchString(value) {
		return 0, fmt.Errorf("%w: expected a decimal amount with at most two decimal places and no grouping separators", models.ErrReceiptResponse)
	}

	negative := strings.HasPrefix(value, "-")
	whole, fraction, _ := strings.Cut(strings.TrimPrefix(value, "-"), ".")
	digits := whole + fraction + strings.Repeat("0", 2-len(fraction))
	amount, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || amount > models.MaxAmountMinor {
		return 0, fmt.Errorf("%w: amount exceeds the supported integer range", models.ErrReceiptResponse)
	}

	if negative {
		amount = -amount
	}

	return amount, nil
}
