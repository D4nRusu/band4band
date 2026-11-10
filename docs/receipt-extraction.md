# Receipt extraction endpoint

This milestone extracts an editable draft from a receipt photo. It does not create expenses, assign contributors, change debts, or save images. Mobile camera capture, item assignment, and private receipt storage are later milestones. The existing `chat.sh` remains a manual llama-cli utility; Go does not execute it.

## Start the model server

Install/use a llama.cpp build with `llama-server` and support for your vision model. Use the same Ministral GGUF and matching projector that worked with `chat.sh`. Set `MODEL_PATH` and `MMPROJ_PATH` in your terminal to those actual local file paths, then run:

```bash
llama-server \
  -m "$MODEL_PATH" \
  --mmproj "$MMPROJ_PATH" \
  -ngl 99 -c 40960 -fa on --jinja \
  --chat-template-kwargs '{"enable_thinking":false}' \
  --alias receipts \
  --host 127.0.0.1 --port 8081
```

The GPU/context settings above match the existing script; adjust them for the server's available memory if necessary. Keep this process running. Go connects to the model over HTTP rather than loading a model per upload. Use a llama.cpp version that supports multimodal chat completions and JSON-schema response formatting. These capabilities vary by installed build/model and still require a live check with your files. See [multimodal support](https://github.com/ggml-org/llama.cpp/blob/master/docs/multimodal.md) and [server documentation](https://github.com/ggml-org/llama.cpp/tree/master/tools/server).

The model server should remain private to the backend. If Go and llama-server run in separate containers, configure their private network address instead of loopback; do not expose inference directly to the mobile app.

## Configure Go

Add these settings to the existing local `backend/.env`, or export them in the backend terminal:

```dotenv
RECEIPT_LLM_URL=http://127.0.0.1:8081/v1/chat/completions
RECEIPT_LLM_MODEL=receipts
```

The URL is the full chat completions endpoint. The model value matches the `--alias receipts` argument above. Restart Go with the existing Firebase credentials configured:

```bash
cd backend
go run .
```

If `RECEIPT_LLM_URL` is unset, the rest of the application works normally and extraction returns 503 for an authorized group member. A configured URL requires a model name; malformed configuration fails startup with a configuration error. No new Go dependencies, Firebase configuration, Storage bucket, or APK build are needed for this endpoint.

## Make a request

`POST /api/groups/{groupID}/receipts/extract` requires the normal Firebase ID token and current membership in that group. Send `multipart/form-data` containing exactly one file part named `image`. Do not send image URLs, local server paths, JSON, or other form fields.

Set `GROUP_ID`, `RECEIPT_PATH`, and `FIREBASE_ID_TOKEN` in your terminal to a real group ID, local JPEG/PNG file path, and current Firebase ID token. The token is the same kind used for the existing API, not a Google ID token. Do not save or commit the token in a file.

```bash
curl --fail-with-body \
  "http://127.0.0.1:8080/api/groups/$GROUP_ID/receipts/extract" \
  -H "Authorization: Bearer $FIREBASE_ID_TOKEN" \
  -F "image=@$RECEIPT_PATH"
```

Example successful response (illustrative receipt):

```json
{
  "currency": "EUR",
  "totalMinor": 1850,
  "itemsTotalMinor": 1850,
  "totalsMatch": true,
  "items": [
    { "id": "item-1", "name": "Pizza", "amountMinor": 1200 },
    { "id": "item-2", "name": "Drink", "amountMinor": 650 }
  ]
}
```

Item IDs identify rows within that draft only; nothing is persisted. Prices are full line totals, not unit prices. Go converts model-provided decimal strings into integer cents without floating-point arithmetic. Individual values and running item totals must fit the application's JavaScript-safe integer range.

The extracted currency must match the group's currency, currently EUR. Missing or unreadable required information produces an error rather than guessing the currency or total. A different item sum is still returned with `totalsMatch: false`, preserving the printed total and item values for later user review. Separately printed discounts can appear as negative draft lines; their eventual allocation rules are outside this milestone. JSON/schema validation does not establish that the image was read accurately.

## Limits and failures

- JPEG or PNG only, validated from the bytes and fully decoded; declared MIME types and filenames are not trusted.
- Maximum image size: 10 MiB. Maximum decoded dimensions: 25 million pixels.
- One active model request per Go process; another request receives 429 with `Retry-After`.
- Model HTTP timeout: 120 seconds. Disconnecting cancels Go's upstream HTTP request; GPU cancellation behavior depends on the model server.
- Maximum model response: 2 MiB; maximum 200 items with names up to 200 characters. Generation is capped at 4096 tokens. Truncated responses fail instead of becoming partial drafts.
- Redirects from the model endpoint are not followed. No receipt images, inference responses, or authentication tokens are logged or written to temporary files.

| Status | Meaning |
| --- | --- |
| 400 | Invalid group ID, multipart upload, image, or decoded dimensions |
| 401 | Missing or invalid Firebase ID token |
| 403 / 404 | Not a current member / group not found |
| 413 | Upload exceeds the image/body limit |
| 422 | Required receipt data unreadable or currency does not match the group |
| 429 | Model already processing another request |
| 502 | Invalid, oversized, or truncated model response |
| 503 | Extraction disabled or model server unavailable/rejected request |
| 504 | Model request timed out |

Responses use `Cache-Control: no-store`. Model error bodies and private upstream details are not returned to callers. The endpoint uses the existing membership read but performs no Firestore writes. Do not automatically retry expensive inference requests in a loop.

## Code and verification

- `internal/httpapi/receipts.go`: authentication-context use, membership checks, upload validation, and error responses.
- `internal/services/receipt_extraction.go`: bounded llama-server HTTP requests, schema, prompt, timeout, and concurrency limit.
- `internal/services/receipt_response.go`: strict response decoding and exact monetary normalization.
- `internal/models/receipt.go`: draft response types and extraction errors.
- `internal/httpapi/routes.go` and `main.go`: route/dependency registration and environment configuration.
- `tests/httpapi/receipts_test.go` and `tests/services/receipt_extraction_test.go`: focused authorization, upload, model, arithmetic, cancellation, and failure tests.

Test execution remains deferred at the user's request. No real receipt, Firebase group, or local model has been exercised as part of implementation. Before connecting the mobile flow, verify a real receipt with the local server, confirm item/total accuracy, check a mismatched or unreadable receipt, and measure latency on the hosting hardware.
