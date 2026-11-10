package httpapi

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

const maxReceiptImageBytes = 10 << 20
const maxReceiptPixels = 25_000_000

var errReceiptUpload = errors.New("upload exactly one JPEG or PNG file in the image form field")
var errReceiptImage = errors.New("the image is damaged or too large to decode; use a clear photo under 25 megapixels")
var errReceiptSize = errors.New("the receipt image must be no larger than 10 MiB")

type ReceiptExtractor interface {
	Extract(context.Context, []byte, string) (models.ReceiptDraft, error)
}

type receiptHandler struct {
	groups    GroupStore
	extractor ReceiptExtractor
}

func (h receiptHandler) extract(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	group, err := h.groups.Details(r.Context(), uid, r.PathValue("groupID"))
	if err != nil {
		writeGroupError(w, err)
		return
	}

	if h.extractor == nil {
		writeReceiptError(w, models.ErrReceiptUnavailable)
		return
	}

	data, contentType, err := readReceiptImage(w, r)
	if err != nil {
		writeReceiptError(w, err)
		return
	}

	draft, err := h.extractor.Extract(r.Context(), data, contentType)
	if err != nil {
		writeReceiptError(w, err)
		return
	}

	if draft.Currency != group.Currency {
		writeGroupJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"code": "receipt_currency_mismatch", "message": "The receipt currency must match the group currency.",
		})
		return
	}

	writeGroupJSON(w, http.StatusOK, draft)
}

func readReceiptImage(w http.ResponseWriter, r *http.Request) ([]byte, string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxReceiptImageBytes+(64<<10))
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, "", errReceiptUpload
	}

	part, err := reader.NextPart()
	if err != nil {
		return nil, "", receiptUploadError(err)
	}
	defer part.Close()

	if part.FormName() != "image" || part.FileName() == "" {
		return nil, "", errReceiptUpload
	}

	data, err := io.ReadAll(io.LimitReader(part, maxReceiptImageBytes+1))
	if err != nil {
		return nil, "", receiptUploadError(err)
	}

	if len(data) > maxReceiptImageBytes {
		return nil, "", errReceiptSize
	}

	if _, err := reader.NextPart(); err != io.EOF {
		return nil, "", receiptUploadError(err)
	}

	contentType := http.DetectContentType(data)
	if contentType != "image/jpeg" && contentType != "image/png" {
		return nil, "", errReceiptUpload
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > maxReceiptPixels/config.Height {
		return nil, "", errReceiptImage
	}

	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return nil, "", errReceiptImage
	}

	return data, contentType, nil
}

func receiptUploadError(err error) error {
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		return errReceiptSize
	}

	return errReceiptUpload
}

func writeReceiptError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	code := "receipt_extraction_failed"
	message := "Could not extract receipt items. Retry with a clearer photo."
	switch {
	case errors.Is(err, errReceiptSize):
		status, code, message = http.StatusRequestEntityTooLarge, "receipt_image_too_large", errReceiptSize.Error()
	case errors.Is(err, errReceiptUpload), errors.Is(err, errReceiptImage):
		status, code, message = http.StatusBadRequest, "invalid_receipt_image", err.Error()
	case errors.Is(err, models.ErrReceiptUnreadable):
		status, code, message = http.StatusUnprocessableEntity, "receipt_unreadable", models.ErrReceiptUnreadable.Error()
	case errors.Is(err, models.ErrReceiptBusy):
		status, code, message = http.StatusTooManyRequests, "receipt_busy", models.ErrReceiptBusy.Error()
		w.Header().Set("Retry-After", "5")
	case errors.Is(err, context.DeadlineExceeded):
		status, code, message = http.StatusGatewayTimeout, "receipt_timeout", "Receipt extraction took too long. Please retry."
	case errors.Is(err, models.ErrReceiptUnavailable):
		status, code, message = http.StatusServiceUnavailable, "receipt_unavailable", "Receipt extraction is unavailable. Check that the model server is configured and running."
	case errors.Is(err, context.Canceled):
		return
	}

	writeGroupJSON(w, status, map[string]string{"code": code, "message": message})
}
