package httpapi_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"firebase.google.com/go/v4/auth"
	"github.com/D4nRusu/band4band/backend/internal/httpapi"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestCreateExpense(t *testing.T) {
	const valid = `{"name":" Lunch ","categoryID":"default_food","paidBy":"other-member","amountMinor":1250}`
	for _, tc := range []struct {
		name, body    string
		authenticated bool
		storeError    error
		status        int
	}{
		{"member pays for another member", valid, true, nil, 201},
		{"unauthenticated", valid, false, nil, 401},
		{"non-member", valid, true, models.ErrGroupAccessForbidden, 403},
		{"payer not in group", valid, true, models.ErrInvalidExpensePayer, 400},
		{"category removed", valid, true, models.ErrInvalidExpenseCategory, 400},
		{"missing group", valid, true, models.ErrGroupNotFound, 404},
		{"zero amount", strings.Replace(valid, "1250", "0", 1), true, nil, 400},
		{"negative amount", strings.Replace(valid, "1250", "-1", 1), true, nil, 400},
		{"decimal amount", strings.Replace(valid, "1250", "12.50", 1), true, nil, 400},
		{"string amount", strings.Replace(valid, "1250", `"1250"`, 1), true, nil, 400},
		{"overflow", strings.Replace(valid, "1250", "9223372036854775808", 1), true, nil, 400},
		{"unsafe JSON integer", strings.Replace(valid, "1250", "9007199254740992", 1), true, nil, 400},
		{"injected creator", strings.Replace(valid, "}", `,"createdBy":"attacker"}`, 1), true, nil, 400},
		{"injected currency", strings.Replace(valid, "}", `,"currency":"USD"}`, 1), true, nil, 400},
		{"injected category label", strings.Replace(valid, "}", `,"categoryName":"Fake"}`, 1), true, nil, 400},
		{"null", "null", true, nil, 400},
		{"trailing JSON", valid + " {}", true, nil, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			store := groupStoreStub{createExpense: func(_ context.Context, uid, groupID string, input models.ExpenseInput) (models.Expense, error) {
				called = true
				if uid != "member" || groupID != "group-id" || input.Name != "Lunch" || input.PaidBy != "other-member" || input.AmountMinor != 1250 {
					t.Fatalf("uid=%s group=%s input=%+v", uid, groupID, input)
				}

				return models.Expense{ID: "expense-id", GroupID: groupID, CreatedBy: uid, AmountMinor: input.AmountMinor, Currency: "EUR"}, tc.storeError
			}}
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
			r := httptest.NewRequest("POST", "/api/groups/group-id/expenses?uid=attacker", strings.NewReader(tc.body))
			if tc.authenticated {
				r.Header.Set("Authorization", "Bearer token")
			}

			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}

			if (tc.status == 401 || (tc.status == 400 && tc.storeError == nil)) && called {
				t.Fatal("invalid request reached storage")
			}

			if tc.status == 201 && (!strings.Contains(w.Body.String(), `"amountMinor":1250`) || w.Header().Get("Cache-Control") != "no-store") {
				t.Fatalf("unexpected response=%s", w.Body.String())
			}
		})
	}
}

func TestExpenseListAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name          string
		authenticated bool
		err           error
		status        int
	}{
		{"member", true, nil, 200},
		{"non-member", true, models.ErrGroupAccessForbidden, 403},
		{"missing group", true, models.ErrGroupNotFound, 404},
		{"unauthenticated", false, nil, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := groupStoreStub{expenses: func(_ context.Context, uid, id string) ([]models.Expense, error) {
				if uid != "member" || id != "group-id" {
					t.Fatalf("uid=%s id=%s", uid, id)
				}
				return nil, tc.err
			}}
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
			r := httptest.NewRequest("GET", "/api/groups/group-id/expenses?uid=outsider", nil)
			if tc.authenticated {
				r.Header.Set("Authorization", "Bearer token")
			}

			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d", w.Code)
			}

			if tc.status == 200 && strings.TrimSpace(w.Body.String()) != "[]" {
				t.Fatalf("body=%s", w.Body.String())
			}
		})
	}
}
