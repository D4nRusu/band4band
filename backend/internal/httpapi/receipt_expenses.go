package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func (h groupHandler) createReceiptExpense(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groupID := r.PathValue("groupID")
	group, err := h.groups.Details(r.Context(), uid, groupID)
	if err != nil {
		writeExpenseError(w, err)
		return
	}

	var input models.ReceiptExpenseInput
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeGroupRequestError(w, "Expected expense details, the receipt currency and total, and reviewed item prices with unit assignments.")
		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		writeGroupRequestError(w, "Expected a single JSON object.")
		return
	}

	expenseInput, err := input.Expense(group.Currency)
	if err != nil {
		writeGroupJSON(w, http.StatusBadRequest, map[string]string{"code": "invalid_receipt_expense", "message": err.Error()})
		return
	}

	expense, err := h.groups.CreateExpense(r.Context(), uid, groupID, expenseInput)
	if err != nil {
		writeExpenseError(w, err)
		return
	}

	expense.ViewerPosition = expense.PositionFor(uid)
	writeGroupJSON(w, http.StatusCreated, expense)
}
