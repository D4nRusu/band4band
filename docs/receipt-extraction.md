# Receipt extraction endpoint

This milestone extracts a draft from a receipt photo and displays it in the mobile app. It does not create expenses, assign contributors, change debts, or save images in cloud storage. Draft editing, item assignment, and private receipt storage are later milestones. The existing `chat.sh` remains a manual llama-cli utility; Go does not execute it.

## Start the model server

Install/use a llama.cpp build with `llama-server` on PATH and support for your vision model. Use the same Ministral GGUF and matching projector that worked with `chat.sh`.

Configure their paths once in the existing `backend/.env` using `RECEIPT_LLM_MODEL_PATH` and `RECEIPT_LLM_MMPROJ_PATH`. The entries are deliberately blank until you supply your actual paths. See `backend/.env.example` for all supported settings; merge them into your existing file rather than overwriting Firebase configuration. Quote paths containing spaces. The launcher reads this file as Bash assignments (`KEY=value`). Absolute paths are recommended; relative paths resolve from `backend/`, and quoted `~` is not expanded.

Then, from the repository root, start the model with:

```bash
./scripts/start-llm.sh
```

No exports or model-path arguments are needed in subsequent terminals. The launcher finds `backend/.env` relative to its own location, so invoking it by its absolute path works from any directory. It runs in the foreground; keep the terminal open and use Ctrl+C to stop it.

The defaults are GPU layers 99, context 40960, thinking off, model alias `receipts`, and loopback port 8082. Optional `RECEIPT_LLM_GPU_LAYERS`, `RECEIPT_LLM_CONTEXT`, and `RECEIPT_LLM_PORT` settings override these values; `RECEIPT_LLM_MODEL` controls the alias used by both Go and the launcher. If you change the port, update `RECEIPT_LLM_URL` to match. The launcher validates the required paths and executable before starting. It does not download models or start Go/Expo.

The GPU/context defaults match the existing script; adjust them for the server's available memory if necessary. Go connects to the model over HTTP rather than loading a model per upload. Use a llama.cpp version that supports multimodal chat completions and JSON-schema response formatting. These capabilities vary by installed build/model and still require a live check with your files. See [multimodal support](https://github.com/ggml-org/llama.cpp/blob/master/docs/multimodal.md) and [server documentation](https://github.com/ggml-org/llama.cpp/tree/master/tools/server).

The model server should remain private to the backend. If Go and llama-server run in separate containers, configure their private network address instead of loopback; do not expose inference directly to the mobile app.

## Configure Go

Keep these settings in the same local `backend/.env`; Go loads them automatically when started from `backend/`:

```dotenv
RECEIPT_LLM_URL=http://127.0.0.1:8082/v1/chat/completions
RECEIPT_LLM_MODEL=receipts
```

The URL is the full chat completions endpoint. The model value matches the `--alias receipts` argument above. Restart Go with the existing Firebase credentials configured:

```bash
cd backend
go run .
```

If `RECEIPT_LLM_URL` is unset, the rest of the application works normally and extraction returns 503 for an authorized group member. A configured URL requires a model name; malformed configuration fails startup with a configuration error. The Go endpoint needs no new Go dependencies, Firebase configuration, or Storage bucket. The mobile photo picker does require the APK rebuild described below.

## Scan from the Android app

The group page now has a **From receipt** button beside Add expense. Open it, take a photo or choose an existing image, review the preview, and tap **Scan receipt**. The app retrieves the Firebase ID token from the signed-in session and uploads the image to Go automatically. No debugger, copied token, or curl command is needed.

The approved `expo-image-picker` dependency adds native code, so rebuild and reinstall the development APK once before using this feature:

```bash
cd mobile
npx eas-cli@latest build --platform android --profile development
```

Install the resulting APK on the phone, then start Metro from `mobile/`:

```bash
npx expo start --dev-client
```

Run the model launcher and Go in their own terminals as described above. Keep the model on port 8082; Metro uses 8081, and Go uses 8080. Make sure both `RECEIPT_LLM_PORT` and the port in `RECEIPT_LLM_URL` in your local `backend/.env` are 8082. These variables never go in the mobile environment.

For USB development with the existing `EXPO_PUBLIC_API_BASE_URL=http://127.0.0.1:8080`, forward Go and Metro:

```bash
adb reverse tcp:8080 tcp:8080
adb reverse tcp:8081 tcp:8081
```

The phone talks only to Go; no forwarding of the model port is needed. On first capture, allow the camera permission. Photo-library selection uses the system chooser; no microphone permission is requested. If a photo is over 10 MiB or 25 megapixels, use the camera's standard-resolution mode or choose a smaller JPEG/PNG.

The result lists item names, quantities, editable line prices, and an editable receipt total. A red message highlights a different item sum. Assign every line to a group member using its dropdown. Whole quantities greater than one offer Split units between people, where each selected member gets a whole number of units; Assign all to one person restores whole-line assignment. Fractional and unknown quantities are assigned as a whole line. Choose an expense name, category, and payer, then confirm creation.

The reviewed line totals must equal the reviewed receipt total. If the receipt and group currencies differ, explicitly confirm using the same numeric amounts in the group currency. This applies no exchange rate. Saving creates a standard unequal expense and updates debts through the existing Go transaction. Receipt rows and photos are not persisted yet.

Cancel scan or the back arrow aborts extraction; closing the modal discards the in-memory draft. While saving, closing, rescanning, and editing are disabled. There is no automatic retry; if a save result is uncertain, close and refresh expenses before retrying. Uploads read the local photo into a real Blob because SDK 57 does not accept legacy URI objects as multipart file parts. Reload the app through Metro without rebuilding the APK. The picker may create local cache files; no manual token files or cloud image records are created.

Check a clear receipt, price corrections, a mismatching/unreadable receipt, whole-line and per-unit assignments, missing assignments, currency confirmation, denied camera permission, choosing an existing image, cancellation, and a stopped model server. Verify the created unequal expense, its debts, ordinary expense creation, and sign-out. Physical-device and live-model verification remains manual.

## Optional command-line request

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
    { "id": "item-1", "name": "Pizza", "quantity": 1, "amountMinor": 1200 },
    { "id": "item-2", "name": "Drink", "quantity": 2, "amountMinor": 650 }
  ]
}
```

Item IDs identify rows within that draft only; nothing is persisted. Preview prices are full line totals. The model reports the printed line total and unit price separately. Go uses the printed line total when available. If the receipt gives only a unit price and quantity, Go multiplies them and rounds to the nearest cent, with half cents rounded away from zero. Lines with a separate line discount need a printed net total to use this preview. Go accepts decimal strings and JSON numbers and calculates money without floating-point arithmetic. Individual values and running item totals must fit the application's JavaScript-safe integer range.

`quantity` is the positive numeric multiplier printed for the line. It may be fractional for weighed items. Go also accepts quoted values such as `"2 buc"` and `"0,75 kg"`, keeping only the number. Missing, unreadable, or invalid quantities are returned as `null` and omitted from the mobile display when the printed line total is available.

Go normalizes `LEI` to `RON`. The extraction response uses the receipt currency even when the group uses a different currency. Missing or unreadable required information produces an error rather than guessing the currency or total. A different item sum is still returned with `totalsMatch: false`, preserving the printed total and item values for user review and correction. Separately printed discounts can appear as negative draft lines and reduce their assignee's final share. JSON/schema validation does not establish that the image was read accurately.

## Creating an expense from reviewed items

Authenticated `POST /api/groups/{groupID}/receipts/expense` accepts `name`, `categoryID`, `paidBy`, `currency`, `confirmCurrency`, `totalMinor`, and `items: [{amountMinor, units, assignments: [{uid, units}]}]`. Prices are reviewed line totals in integer minor units. Whole-line assignments use all of that line's units; fractional or unknown quantities use one assignable portion for the entire line. The request is limited to 256 KiB and 200 lines.

Go validates complete assignments, distinct members within each line, exact totals, and explicit currency confirmation. It splits a line's cents across units and aggregates shares by UID. Any leftover cents go to units assigned to the first UIDs in ascending order. For example, 10.01 across three units gives two units 3.34 each and one unit 3.33. Assigning the first two units to one person gives that person 6.68.

Negative discount lines reduce the assigned person's aggregate share. Each person's final share must remain zero or positive. The existing creation transaction verifies current caller, payer, and contributor membership plus category availability, then saves an unequal expense and updates both debt graphs. Only the aggregate expense is stored; receipt rows and the photo remain temporary.

## Limits and failures

- JPEG or PNG only, validated from the bytes and fully decoded; declared MIME types and filenames are not trusted.
- Maximum image size: 10 MiB. Maximum decoded dimensions: 25 million pixels.
- One active model request per Go process; another request receives 429 with `Retry-After`.
- Model HTTP timeout: 120 seconds. Disconnecting cancels Go's upstream HTTP request; GPU cancellation behavior depends on the model server.
- Maximum model response: 2 MiB; maximum 200 items with names up to 200 characters. Generation is capped at 4096 tokens. Truncated responses fail instead of becoming partial drafts.
- Redirects from the model endpoint are not followed. Receipt images and authentication tokens are never logged or written to temporary files. Model responses are logged only when the opt-in diagnostics below are enabled.

| Status | Meaning |
| --- | --- |
| 400 | Invalid group ID, multipart upload, image, or decoded dimensions |
| 401 | Missing or invalid Firebase ID token |
| 403 / 404 | Not a current member / group not found |
| 413 | Upload exceeds the image/body limit |
| 422 | Required receipt data unreadable |
| 429 | Model already processing another request |
| 502 | Invalid, oversized, or truncated model response |
| 503 | Extraction disabled or model server unavailable/rejected request |
| 504 | Model request timed out |

Responses use `Cache-Control: no-store`. Model error bodies and private upstream details are not returned to callers. The endpoint uses the existing membership read but performs no Firestore writes. Do not automatically retry expensive inference requests in a loop.

## Inspect rejected model responses

Set `RECEIPT_LLM_DEBUG=true` in the local `backend/.env`, restart Go, and scan again. Lines prefixed `[receipt-llm]` in the **Go terminal** show the upstream HTTP status, raw response (limited to 64 KiB), and the precise rejection reason. The raw response includes the model's finish reason and usage fields when supplied. `finish_reason="length"` indicates that the generation hit its token limit; JSON/type errors or item validation failures identify the failing field or row. This does not relax validation or expose internal diagnostics to the mobile client.

Only the model response is recorded, not request headers, Firebase tokens, prompts, or image bytes. The output may contain receipt text. Set `RECEIPT_LLM_DEBUG=false` and restart Go after diagnosis; diagnostics are disabled by default and the example environment keeps them off. No debug files are created.

The mobile `Response.blob()` warning describes extra copying/base64 overhead while reading the photo with React Native's Blob implementation. Once Go and the model receive the image, that warning is not the cause of model-response validation failure. The existing upload remains functional; `expo-blob` is an optional future performance improvement requiring separate dependency approval and a native rebuild, not a prerequisite for inspecting extraction results.

## Code and verification

- `internal/httpapi/receipts.go`: authentication-context use, membership checks, upload validation, and error responses.
- `internal/services/receipt_extraction.go`: bounded llama-server HTTP requests, schema, prompt, timeout, and concurrency limit.
- `internal/services/receipt_response.go`: strict response decoding and exact monetary normalization.
- `internal/models/receipt.go`: draft response types and extraction errors.
- `internal/httpapi/routes.go` and `main.go`: route/dependency registration and environment configuration.
- `tests/httpapi/receipts_test.go` and `tests/services/receipt_extraction_test.go`: focused authorization, upload, model, arithmetic, cancellation, and failure tests.

The assignment milestone adds `backend/internal/models/receipt_expense.go`, `backend/internal/httpapi/receipt_expenses.go`, and model/HTTP tests in `backend/tests/models/receipt_expense_test.go` and `backend/tests/httpapi/receipt_expenses_test.go`. The new endpoint is registered in `backend/internal/httpapi/routes.go`.

Mobile additions are `src/components/receipt-assignment-form.tsx`, `src/components/receipt-item-editor.tsx`, `src/hooks/use-receipt-assignment.ts`, `src/lib/receipt-assignment.ts`, and `tests/receipt-assignment.test.mjs`. Integration updates `receipt-scan-modal.tsx`, `group-details-screen.android.tsx`, `expense-payer-select.tsx`, `use-group-expenses.ts`, and `expenses-api.ts`. The form replaces the former `receipt-draft-details.tsx` component. This guide and `docs/architecture.md` document the new flow.

Go model and HTTP tests, Go build/vet, and the four mobile assignment helper tests passed. Full mobile typecheck and lint still report existing errors in `group-card.tsx` and `use-color-scheme.web.ts`. Creating an assigned receipt expense against Firebase and testing the updated UI on a physical phone remain manual verification steps.
