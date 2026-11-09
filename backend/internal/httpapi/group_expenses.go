package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func (h groupHandler) expenses(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	expenses, err := h.groups.Expenses(r.Context(), uid, r.PathValue("groupID"))
	if err != nil {
		writeExpenseError(w, err)
		return
	}

	if expenses == nil {
		expenses = []models.Expense{}
	}

	writeGroupJSON(w, http.StatusOK, expenses)
}

func (h groupHandler) createExpense(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input models.ExpenseInput
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeGroupRequestError(w, "Expected name, categoryID, paidBy, and an integer amountMinor in cents.")
		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		writeGroupRequestError(w, "Expected a single JSON object.")
		return
	}

	input, err := input.Normalize()
	if err != nil {
		writeExpenseError(w, err)
		return
	}

	expense, err := h.groups.CreateExpense(r.Context(), uid, r.PathValue("groupID"), input)
	if err != nil {
		writeExpenseError(w, err)
		return
	}

	writeGroupJSON(w, http.StatusCreated, expense)
}

func writeExpenseError(w http.ResponseWriter, err error) {
	for _, validation := range []error{
		models.ErrInvalidExpenseID, models.ErrInvalidExpenseName, models.ErrInvalidExpenseAmount, models.ErrInvalidExpenseCategory,
		models.ErrInvalidExpensePayer, models.ErrUnsupportedCurrency,
	} {
		if errors.Is(err, validation) {
			writeGroupJSON(w, http.StatusBadRequest, map[string]string{"code": "invalid_expense", "message": validation.Error()})
			return
		}
	}

	writeGroupError(w, err)
}

func (h groupHandler) deleteExpense(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.groups.DeleteExpense(r.Context(), uid, r.PathValue("groupID"), r.PathValue("expenseID")); err != nil {
		writeExpenseError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
