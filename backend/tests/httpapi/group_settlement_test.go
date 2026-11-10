package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"firebase.google.com/go/v4/auth"
	"github.com/D4nRusu/band4band/backend/internal/httpapi"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestSettleUpEndpoint(t *testing.T) {
	const valid = `{"fromUID":"member","toUID":"bob","amountMinor":500}`
	for _, tc := range []struct {
		name, body    string
		authenticated bool
		storeError    error
		status        int
	}{
		{"member settles with another member", valid, true, nil, 204},
		{"unauthenticated", valid, false, nil, 401},
		{"invalid payload", `{"toUID":"bob"}`, true, nil, 400},
		{"missing payer", `{"toUID":"bob","amountMinor":500}`, true, nil, 400},
		{"not payer or recipient", valid, true, models.ErrSettlementNotParty, 400},
		{"debt too small", valid, true, models.ErrDebtUnderflow, 400},
		{"non-member", valid, true, models.ErrGroupAccessForbidden, 403},
		{"missing group", valid, true, models.ErrGroupNotFound, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			store := groupStoreStub{settleUp: func(_ context.Context, uid, groupID, fromUID, toUID string, amountMinor int64) error {
				called = true
				if uid != "member" || groupID != "group-id" || fromUID != "member" || toUID != "bob" || amountMinor != 500 {
					t.Fatalf("uid=%s group=%s from=%s to=%s amount=%d", uid, groupID, fromUID, toUID, amountMinor)
				}

				return tc.storeError
			}}
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
			r := httptest.NewRequest("POST", "/api/groups/group-id/settlements?uid=attacker", strings.NewReader(tc.body))
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
		})
	}
}

func TestUserOwedSummaryEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name          string
		authenticated bool
		err           error
		status        int
	}{
		{"member", true, nil, 200},
		{"non-member", true, models.ErrGroupAccessForbidden, 403},
		{"invalid requested user", true, models.ErrInvalidDebtUser, 400},
		{"missing group", true, models.ErrGroupNotFound, 404},
		{"unauthenticated", false, nil, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := groupStoreStub{userOwed: func(_ context.Context, uid, groupID, userUID string) (models.UserOwedSummary, error) {
				if uid != "member" || groupID != "group-id" || userUID != "alice" {
					t.Fatalf("uid=%s group=%s user=%s", uid, groupID, userUID)
				}

				return models.UserOwedSummary{
					UserUID:          "alice",
					Currency:         "EUR",
					TotalAmountMinor: 900,
					Debts:            []models.UserOwedDebt{{ToUID: "bob", AmountMinor: 500}, {ToUID: "charlie", AmountMinor: 400}},
				}, tc.err
			}}
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
			r := httptest.NewRequest("GET", "/api/groups/group-id/users/alice/owed?uid=outsider", nil)
			if tc.authenticated {
				r.Header.Set("Authorization", "Bearer token")
			}

			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}

			if tc.status == 200 {
				var summary models.UserOwedSummary
				if err := json.Unmarshal(w.Body.Bytes(), &summary); err != nil {
					t.Fatal(err)
				}

				if summary.TotalAmountMinor != 900 || len(summary.Debts) != 2 {
					t.Fatalf("summary=%+v", summary)
				}
			}
		})
	}
}

func TestUserReceivableSummaryEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name          string
		authenticated bool
		err           error
		status        int
	}{
		{"member", true, nil, 200},
		{"non-member", true, models.ErrGroupAccessForbidden, 403},
		{"invalid requested user", true, models.ErrInvalidDebtUser, 400},
		{"missing group", true, models.ErrGroupNotFound, 404},
		{"unauthenticated", false, nil, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := groupStoreStub{userReceivable: func(_ context.Context, uid, groupID, userUID string) (models.UserReceivableSummary, error) {
				if uid != "member" || groupID != "group-id" || userUID != "alice" {
					t.Fatalf("uid=%s group=%s user=%s", uid, groupID, userUID)
				}

				return models.UserReceivableSummary{
					UserUID:          "alice",
					Currency:         "EUR",
					TotalAmountMinor: 1100,
					Debts: []models.UserReceivableDebt{
						{FromUID: "bob", AmountMinor: 700},
						{FromUID: "charlie", AmountMinor: 400},
					},
				}, tc.err
			}}
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
			r := httptest.NewRequest("GET", "/api/groups/group-id/users/alice/receivable?uid=outsider", nil)
			if tc.authenticated {
				r.Header.Set("Authorization", "Bearer token")
			}

			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}

			if tc.status == 200 {
				var summary models.UserReceivableSummary
				if err := json.Unmarshal(w.Body.Bytes(), &summary); err != nil {
					t.Fatal(err)
				}

				if summary.TotalAmountMinor != 1100 || len(summary.Debts) != 2 {
					t.Fatalf("summary=%+v", summary)
				}
			}
		})
	}
}
