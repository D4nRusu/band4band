package services_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/models"
	"github.com/D4nRusu/band4band/backend/internal/services"
)

func TestReceiptDiagnosticsAreOptInAndExplainRejection(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		completionResponse(w, readableReceipt, "length")
	}))
	defer server.Close()
	extractor, err := services.NewReceiptExtractor(server.URL, "receipts")
	if err != nil {
		t.Fatal(err)
	}

	photo := []byte("private photo bytes must not be logged")
	if _, err := extractor.Extract(context.Background(), photo, "image/png"); !errors.Is(err, models.ErrReceiptResponse) {
		t.Fatalf("err=%v", err)
	}

	if output.Len() != 0 {
		t.Fatal("receipt diagnostics were enabled by default")
	}

	extractor.Debug = true
	if _, err := extractor.Extract(context.Background(), photo, "image/png"); !errors.Is(err, models.ErrReceiptResponse) {
		t.Fatalf("err=%v", err)
	}

	for _, expected := range []string{"[receipt-llm] HTTP 200", `"choices"`, "Pizza", `finish_reason="length"`, "token limit"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("missing %q in diagnostics", expected)
		}
	}

	if strings.Contains(output.String(), base64.StdEncoding.EncodeToString(photo)) || strings.Contains(output.String(), "Extract only information visible") {
		t.Fatal("request image or prompt was logged")
	}
}

func TestReceiptDiagnosticResponseIsBounded(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("x", 70<<10)))
	}))
	defer server.Close()
	extractor, err := services.NewReceiptExtractor(server.URL, "receipts")
	if err != nil {
		t.Fatal(err)
	}

	extractor.Debug = true
	if _, err := extractor.Extract(context.Background(), nil, "image/png"); !errors.Is(err, models.ErrReceiptResponse) {
		t.Fatalf("err=%v", err)
	}

	if !strings.Contains(output.String(), "showing 65536 of 71680 bytes") || output.Len() > (64<<10)+1024 {
		t.Fatal("model response logging exceeded its bound")
	}
}

func TestReceiptRejectionIdentifiesInvalidField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		completionResponse(w, strings.Replace(readableReceipt, `"6.50"`, `"6.501"`, 1), "stop")
	}))
	defer server.Close()
	extractor, err := services.NewReceiptExtractor(server.URL, "receipts")
	if err != nil {
		t.Fatal(err)
	}

	_, err = extractor.Extract(context.Background(), nil, "image/png")
	if !errors.Is(err, models.ErrReceiptResponse) || !strings.Contains(err.Error(), "item 2 amount") || !strings.Contains(err.Error(), "two decimal places") {
		t.Fatalf("missing field-level diagnostic: %v", err)
	}
}
