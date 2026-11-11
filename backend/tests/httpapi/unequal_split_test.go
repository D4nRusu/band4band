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

func TestUnequalExpenseRequest(t *testing.T) {
	const valid = `{"name":"Dinner","categoryID":"default_food","paidBy":"member","amountMinor":1000,"splitType":"unequal","contributions":[{"uid":"member","amountMinor":300},{"uid":"other","amountMinor":700}]}`
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"unequal shares", valid, 201},
		{"does not add up", strings.Replace(valid, ":700", ":699", 1), 400},
		{"negative share", strings.Replace(valid, ":300", ":-300", 1), 400},
		{"fractional cents", strings.Replace(valid, ":300", ":300.5", 1), 400},
		{"duplicate contributor", strings.Replace(valid, `"uid":"other"`, `"uid":"member"`, 1), 400},
		{"client-supplied name", strings.Replace(valid, `"uid":"other"`, `"uid":"other","name":"Fake"`, 1), 400},
		{"unknown mode", strings.Replace(valid, `"unequal"`, `"percentage"`, 1), 400},
		{"conflicting input", strings.Replace(valid, `"splitType"`, `"participantUIDs":["member"],"splitType"`, 1), 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			store := groupStoreStub{createExpense: func(_ context.Context, uid, groupID string, input models.ExpenseInput) (models.Expense, error) {
				called = true
				if uid != "member" || groupID != "group-id" || input.SplitType != "unequal" || len(input.Contributions) != 2 {
					t.Fatalf("input=%+v", input)
				}

				return models.Expense{SplitType: input.SplitType, AmountMinor: input.AmountMinor}, nil
			}}
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
				return &auth.Token{UID: "member"}, nil
			})
			r := httptest.NewRequest("POST", "/api/groups/group-id/expenses", strings.NewReader(tc.body))
			r.Header.Set("Authorization", "Bearer token")
			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil, nil).ServeHTTP(w, r)
			if w.Code != tc.status || called != (tc.status == 201) {
				t.Fatalf("status=%d called=%v body=%s", w.Code, called, w.Body.String())
			}
		})
	}
}
