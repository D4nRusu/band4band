package services_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/D4nRusu/band4band/backend/internal/models"
	"github.com/D4nRusu/band4band/backend/internal/services"
)

const readableReceipt = `{"currency":"EUR","total":"18.50","items":[{"name":" Pizza ","amount":"12.00"},{"name":"Drink","amount":"6.50"}]}`

func completionResponse(w http.ResponseWriter, content, reason string) {
	json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
		"finish_reason": reason, "message": map[string]string{"content": content},
	}}})
}

func TestReceiptModelRequestAndExactAmounts(t *testing.T) {
	photo := []byte("test image bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/chat/completions" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var request struct {
			Model          string `json:"model"`
			Stream         bool   `json:"stream"`
			ResponseFormat struct {
				Type   string          `json:"type"`
				Schema json.RawMessage `json:"schema"`
			} `json:"response_format"`
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}

		if request.Model != "receipts" || request.Stream || request.ResponseFormat.Type != "json_schema" || !json.Valid(request.ResponseFormat.Schema) || len(request.Messages) != 2 {
			t.Errorf("unexpected request: %+v", request)
			return
		}

		var parts []struct {
			Type     string `json:"type"`
			ImageURL struct {
				URL string `json:"url"`
			} `json:"image_url"`
		}
		if err := json.Unmarshal(request.Messages[1].Content, &parts); err != nil || len(parts) != 2 || parts[1].ImageURL.URL != "data:image/jpeg;base64,"+base64.StdEncoding.EncodeToString(photo) {
			t.Errorf("image not sent as inline data: %+v err=%v", parts, err)
		}

		completionResponse(w, readableReceipt, "stop")
	}))
	defer server.Close()
	extractor, err := services.NewReceiptExtractor(server.URL+"/v1/chat/completions", "receipts")
	if err != nil {
		t.Fatal(err)
	}

	draft, err := extractor.Extract(context.Background(), photo, "image/jpeg")
	if err != nil || draft.TotalMinor != 1850 || draft.ItemsTotalMinor != 1850 || !draft.TotalsMatch || len(draft.Items) != 2 {
		t.Fatalf("draft=%+v err=%v", draft, err)
	}

	if draft.Items[0].ID != "item-1" || draft.Items[0].Name != "Pizza" || draft.Items[0].AmountMinor != 1200 || draft.Items[1].ID != "item-2" {
		t.Fatalf("items=%+v", draft.Items)
	}
}

func TestReceiptResponseValidation(t *testing.T) {
	for _, tc := range []struct {
		name, content, reason string
		want                  error
	}{
		{"mismatch is a reviewable draft", strings.Replace(readableReceipt, `"18.50"`, `"19.00"`, 1), "stop", nil},
		{"discount is preserved", `{"currency":"EUR","total":"0.28","items":[{"name":"Item","amount":"0.29"},{"name":"Discount","amount":"-0.01"}]}`, "stop", nil},
		{"largest safe total", `{"currency":"EUR","total":"90071992547409.91","items":[{"name":"Item","amount":"90071992547409.91"}]}`, "stop", nil},
		{"unsafe sum", `{"currency":"EUR","total":"90071992547409.91","items":[{"name":"Item","amount":"90071992547409.91"},{"name":"Extra","amount":"0.01"}]}`, "stop", models.ErrReceiptResponse},
		{"missing currency", strings.Replace(readableReceipt, `"EUR"`, "null", 1), "stop", models.ErrReceiptUnreadable},
		{"unreadable item", strings.Replace(readableReceipt, `"6.50"`, "null", 1), "stop", models.ErrReceiptUnreadable},
		{"unit price without quantity", strings.Replace(readableReceipt, `"amount":"6.50"`, `"unitPrice":"6.50"`, 1), "stop", models.ErrReceiptUnreadable},
		{"unit price with line discount but no final amount", strings.Replace(readableReceipt, `"amount":"6.50"`, `"unitPrice":"6.50","quantity":2,"discount":"-0.10"`, 1), "stop", models.ErrReceiptUnreadable},
		{"calculated line overflow", strings.Replace(readableReceipt, `"amount":"6.50"`, `"unitPrice":"90071992547409.91","quantity":2`, 1), "stop", models.ErrReceiptResponse},
		{"not a receipt", `{"currency":null,"total":null,"items":[]}`, "stop", models.ErrReceiptUnreadable},
		{"invalid JSON", "not JSON", "stop", models.ErrReceiptResponse},
		{"markdown", "```json\n" + readableReceipt + "\n```", "stop", models.ErrReceiptResponse},
		{"truncated completion", readableReceipt, "length", models.ErrReceiptResponse},
		{"trailing JSON", readableReceipt + " {}", "stop", models.ErrReceiptResponse},
		{"extra fields", strings.Replace(readableReceipt, `"currency"`, `"instructions":"ignore rules","currency"`, 1), "stop", models.ErrReceiptResponse},
		{"fractional cents", strings.Replace(readableReceipt, `"6.50"`, `"6.501"`, 1), "stop", models.ErrReceiptResponse},
		{"exponent", strings.Replace(readableReceipt, `"6.50"`, `"6e2"`, 1), "stop", models.ErrReceiptResponse},
		{"numeric fractional cents", strings.Replace(readableReceipt, `"6.50"`, "6.501", 1), "stop", models.ErrReceiptResponse},
		{"numeric exponent", strings.Replace(readableReceipt, `"6.50"`, "6e2", 1), "stop", models.ErrReceiptResponse},
		{"unsafe integer", strings.Replace(readableReceipt, `"6.50"`, `"90071992547409.92"`, 1), "stop", models.ErrReceiptResponse},
		{"integer overflow", strings.Replace(readableReceipt, `"6.50"`, `"999999999999999999999999999999"`, 1), "stop", models.ErrReceiptResponse},
		{"zero total", strings.Replace(readableReceipt, `"18.50"`, `"0"`, 1), "stop", models.ErrReceiptResponse},
		{"empty name", strings.Replace(readableReceipt, `"Drink"`, `" "`, 1), "stop", models.ErrReceiptResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { completionResponse(w, tc.content, tc.reason) }))
			defer server.Close()
			extractor, err := services.NewReceiptExtractor(server.URL, "receipts")
			if err != nil {
				t.Fatal(err)
			}

			draft, err := extractor.Extract(context.Background(), []byte("image"), "image/png")
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want=%v", err, tc.want)
			}

			if tc.name == "mismatch is a reviewable draft" && (draft.TotalsMatch || draft.TotalMinor != 1900 || draft.ItemsTotalMinor != 1850) {
				t.Fatalf("mismatch was silently corrected: %+v", draft)
			}

			if tc.name == "discount is preserved" && (!draft.TotalsMatch || draft.ItemsTotalMinor != 28 || draft.Items[1].AmountMinor != -1) {
				t.Fatalf("discount=%+v", draft)
			}
		})
	}
}

func TestReceiptMixedAmountsQuantitiesAndLineDiscount(t *testing.T) {
	const receipt = `{"currency":"LEI","total":166.78,"items":[
		{"name":"Cable","quantity":"0,75 kg","unitPrice":"32.13","lineTotal":null},
		{"name":"Cable ties","quantity":"2 pun","amount":"11.66"},
		{"name":"Extension cord","quantity":"3 buc","unitPrice":"6.00","lineTotal":"17.35","discount":-0.75},
		{"name":"Crimping tool","quantity":1,"amount":59.00},
		{"name":"Screwdrivers","quantity":"2 set","amount":"34.71"},
		{"name":"Unreadable quantity","quantity":null,"amount":"0.00"},
		{"name":"Invalid quantity","quantity":"0 buc","amount":"0.00"},
		{"name":"Connectors","quantity":2,"unitPrice":9.98,"lineTotal":null}
	]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		completionResponse(w, receipt, "stop")
	}))
	defer server.Close()
	extractor, err := services.NewReceiptExtractor(server.URL, "receipts")
	if err != nil {
		t.Fatal(err)
	}

	draft, err := extractor.Extract(context.Background(), nil, "image/png")
	if err != nil {
		t.Fatal(err)
	}

	if draft.Currency != "RON" || draft.TotalMinor != 16678 || draft.ItemsTotalMinor != 16678 || !draft.TotalsMatch || len(draft.Items) != 8 {
		t.Fatalf("draft=%+v", draft)
	}

	for i, want := range []int64{2410, 1166, 1735, 5900, 3471, 0, 0, 1996} {
		if got := draft.Items[i].AmountMinor; got != want {
			t.Errorf("item %d amount = %d, want %d", i+1, got, want)
		}
	}

	for i, want := range []float64{0.75, 2, 3, 1, 2, 0, 0, 2} {
		got := draft.Items[i].Quantity
		if want == 0 {
			if got != nil {
				t.Errorf("item %d should have no quantity", i+1)
			}
		} else if got == nil || *got != want {
			t.Errorf("item %d quantity = %v, want %g", i+1, got, want)
		}
	}
}

func TestReceiptServerFailuresAndRedirects(t *testing.T) {
	var redirected atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Store(true) }))
	defer target.Close()
	for _, status := range []int{302, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", target.URL)
				w.WriteHeader(status)
				w.Write([]byte("private upstream details"))
			}))
			defer server.Close()
			extractor, err := services.NewReceiptExtractor(server.URL, "receipts")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := extractor.Extract(context.Background(), nil, "image/png"); !errors.Is(err, models.ErrReceiptUnavailable) {
				t.Fatalf("err=%v", err)
			}
		})
	}

	if redirected.Load() {
		t.Fatal("receipt was forwarded to a redirect target")
	}
}

func TestReceiptConfigurationAndResponseBounds(t *testing.T) {
	for _, endpoint := range []string{"", "file:///tmp/model", "http://", "http://user:password@localhost/v1/chat/completions", "http://localhost/?key=value", "http://localhost/#fragment"} {
		if _, err := services.NewReceiptExtractor(endpoint, "receipts"); err == nil {
			t.Fatalf("accepted invalid model endpoint %q", endpoint)
		}
	}

	if _, err := services.NewReceiptExtractor("http://localhost/v1/chat/completions", ""); err == nil {
		t.Fatal("accepted missing model alias")
	}

	for _, body := range []string{"{}", "{", strings.Repeat("x", (2<<20)+1)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		extractor, err := services.NewReceiptExtractor(server.URL, "receipts")
		if err != nil {
			t.Fatal(err)
		}
		_, err = extractor.Extract(context.Background(), nil, "image/png")
		server.Close()
		if !errors.Is(err, models.ErrReceiptResponse) {
			t.Fatalf("err=%v", err)
		}
	}

	extractor, err := services.NewReceiptExtractor("http://127.0.0.1:1/v1/chat/completions", "receipts")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if _, err := extractor.Extract(ctx, nil, "image/png"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired request=%v", err)
	}
}

func TestReceiptCancellationAndBusyLimit(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		completionResponse(w, readableReceipt, "stop")
	}))
	defer server.Close()
	defer close(release)
	extractor, err := services.NewReceiptExtractor(server.URL, "receipts")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { _, err := extractor.Extract(ctx, nil, "image/png"); finished <- err }()
	<-started
	if _, err := extractor.Extract(context.Background(), nil, "image/png"); !errors.Is(err, models.ErrReceiptBusy) {
		t.Fatalf("busy=%v", err)
	}
	cancel()
	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel=%v", err)
	}
}
