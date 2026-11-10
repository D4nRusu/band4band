package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"firebase.google.com/go/v4/auth"
	"github.com/D4nRusu/band4band/backend/internal/httpapi"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

type receiptExtractorFunc func(context.Context, []byte, string) (models.ReceiptDraft, error)

func (f receiptExtractorFunc) Extract(ctx context.Context, data []byte, mime string) (models.ReceiptDraft, error) {
	return f(ctx, data, mime)
}

func receiptPhoto(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func receiptUpload(t *testing.T, data []byte, field string, extraPart bool) (*bytes.Buffer, string) {
	t.Helper()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(field, "receipt.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if extraPart {
		if err := writer.WriteField("extra", "unexpected"); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body, writer.FormDataContentType()
}

func TestReceiptEndpointAuthorizationAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		authenticated, configured bool
		groupError, modelError    error
		currency                  string
		status                    int
	}{
		{"member", true, true, nil, nil, "EUR", 200},
		{"unauthenticated", false, true, nil, nil, "EUR", 401},
		{"non-member", true, true, models.ErrGroupAccessForbidden, nil, "EUR", 403},
		{"missing group", true, true, models.ErrGroupNotFound, nil, "EUR", 404},
		{"invalid group", true, true, models.ErrInvalidGroupID, nil, "EUR", 400},
		{"not configured", true, false, nil, nil, "EUR", 503},
		{"unavailable", true, true, nil, models.ErrReceiptUnavailable, "EUR", 503},
		{"unreadable", true, true, nil, models.ErrReceiptUnreadable, "EUR", 422},
		{"model failure", true, true, nil, errors.New("private upstream details"), "EUR", 502},
		{"timeout", true, true, nil, context.DeadlineExceeded, "EUR", 504},
		{"busy", true, true, nil, models.ErrReceiptBusy, "EUR", 429},
		{"different currency", true, true, nil, nil, "USD", 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			groupCalled, modelCalled := false, false
			store := groupStoreStub{details: func(_ context.Context, uid, id string) (models.Group, error) {
				groupCalled = true
				if uid != "member" || id != "group-id" {
					t.Fatalf("uid=%s group=%s", uid, id)
				}
				return models.Group{Currency: "EUR"}, tc.groupError
			}}
			var extractor httpapi.ReceiptExtractor
			photo := receiptPhoto(t)
			if tc.configured {
				extractor = receiptExtractorFunc(func(ctx context.Context, data []byte, mime string) (models.ReceiptDraft, error) {
					modelCalled = true
					if !bytes.Equal(data, photo) || mime != "image/png" {
						t.Fatal("image bytes or detected MIME changed")
					}
					return models.ReceiptDraft{Currency: tc.currency, TotalMinor: 150, ItemsTotalMinor: 100, TotalsMatch: false,
						Items: []models.ReceiptItem{{ID: "item-1", Name: "Food", AmountMinor: 100}}}, tc.modelError
				})
			}
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
			body, contentType := receiptUpload(t, photo, "image", false)
			r := httptest.NewRequest("POST", "/api/groups/group-id/receipts/extract?uid=outsider", body)
			r.Header.Set("Content-Type", contentType)
			if tc.authenticated {
				r.Header.Set("Authorization", "Bearer token")
			}
			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil, extractor).ServeHTTP(w, r)
			if w.Code != tc.status || groupCalled != tc.authenticated || modelCalled != (tc.authenticated && tc.groupError == nil && tc.configured) {
				t.Fatalf("status=%d group=%v model=%v body=%s", w.Code, groupCalled, modelCalled, w.Body.String())
			}

			if strings.Contains(w.Body.String(), "private upstream details") {
				t.Fatal("upstream details leaked")
			}
			if tc.status != 401 && w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("missing no-store header")
			}
			if tc.status == 429 && w.Header().Get("Retry-After") == "" {
				t.Fatal("missing retry delay")
			}
			if tc.status == 200 {
				var draft models.ReceiptDraft
				if err := json.Unmarshal(w.Body.Bytes(), &draft); err != nil {
					t.Fatal(err)
				}
				if draft.TotalsMatch || draft.TotalMinor != 150 || draft.ItemsTotalMinor != 100 {
					t.Fatalf("mismatch lost: %+v", draft)
				}
			}
		})
	}
}

func TestReceiptImageValidation(t *testing.T) {
	photo := receiptPhoto(t)
	for _, tc := range []struct {
		name, field string
		data        []byte
		extra       bool
		status      int
	}{
		{"empty file", "image", nil, false, 400},
		{"fake image", "image", []byte("not an image"), false, 400},
		{"truncated PNG", "image", photo[:33], false, 400},
		{"wrong field", "file", photo, false, 400},
		{"multiple parts", "image", photo, true, 400},
		{"oversized", "image", bytes.Repeat([]byte("x"), (10<<20)+1), false, 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := groupStoreStub{details: func(context.Context, string, string) (models.Group, error) { return models.Group{Currency: "EUR"}, nil }}
			extractor := receiptExtractorFunc(func(context.Context, []byte, string) (models.ReceiptDraft, error) {
				t.Fatal("invalid upload reached model")
				return models.ReceiptDraft{}, nil
			})
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
			body, contentType := receiptUpload(t, tc.data, tc.field, tc.extra)
			r := httptest.NewRequest("POST", "/api/groups/group-id/receipts/extract", body)
			r.Header.Set("Authorization", "Bearer token")
			r.Header.Set("Content-Type", contentType)
			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil, extractor).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
