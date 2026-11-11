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

func TestReceiptExpenseEndpoint(t *testing.T) {
	const valid = `{"name":"Receipt","categoryID":"default_other","paidBy":"payer","currency":"EUR","totalMinor":1996,"items":[{"amountMinor":1996,"units":2,"assignments":[{"uid":"b","units":1},{"uid":"a","units":1}]}]}`
	for _, tc := range []struct {
		name, body    string
		authenticated bool
		groupError    error
		createError   error
		status        int
		writes        bool
	}{
		{"creates unequal expense", valid, true, nil, nil, 201, true},
		{"unauthenticated", valid, false, nil, nil, 401, false},
		{"non-member", valid, true, models.ErrGroupAccessForbidden, nil, 403, false},
		{"contributor left", valid, true, nil, models.ErrInvalidExpenseParticipants, 400, true},
		{"payer left", valid, true, nil, models.ErrInvalidExpensePayer, 400, true},
		{"incomplete allocation", strings.Replace(valid, `"units":2`, `"units":3`, 1), true, nil, nil, 400, false},
		{"wrong total", strings.Replace(valid, `"totalMinor":1996`, `"totalMinor":1995`, 1), true, nil, nil, 400, false},
		{"unconfirmed currency", strings.Replace(valid, `"EUR"`, `"RON"`, 1), true, nil, nil, 400, false},
		{"confirmed currency", strings.Replace(valid, `"currency":"EUR"`, `"currency":"RON","confirmCurrency":true`, 1), true, nil, nil, 201, true},
		{"injected shares", strings.Replace(valid, `"name":`, `"contributions":[],"name":`, 1), true, nil, nil, 400, false},
		{"fractional units", strings.Replace(valid, `"units":2`, `"units":2.5`, 1), true, nil, nil, 400, false},
		{"trailing json", valid + " {}", true, nil, nil, 400, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			store := groupStoreStub{
				details: func(_ context.Context, uid, id string) (models.Group, error) {
					if uid != "member" || id != "group-id" {
						t.Fatalf("uid=%s id=%s", uid, id)
					}
					return models.Group{Currency: "EUR"}, tc.groupError
				},
				createExpense: func(_ context.Context, uid, id string, input models.ExpenseInput) (models.Expense, error) {
					called = true
					if uid != "member" || id != "group-id" || input.SplitType != "unequal" || input.AmountMinor != 1996 ||
						len(input.Contributions) != 2 || input.Contributions[0].AmountMinor != 998 || input.Contributions[1].AmountMinor != 998 {
						t.Fatalf("uid=%s id=%s input=%+v", uid, id, input)
					}
					return models.Expense{ID: "expense", AmountMinor: input.AmountMinor}, tc.createError
				},
			}
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
			request := httptest.NewRequest("POST", "/api/groups/group-id/receipts/expense", strings.NewReader(tc.body))
			if tc.authenticated {
				request.Header.Set("Authorization", "Bearer test-token")
			}
			response := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil, nil).ServeHTTP(response, request)
			if response.Code != tc.status || called != tc.writes {
				t.Fatalf("status=%d called=%t body=%s", response.Code, called, response.Body.String())
			}
		})
	}
}
