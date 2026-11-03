# PAAM — Architecture

## Overview

PAAM is a mobile application for tracking shared expenses, individual debts, and settlements between users.

## Components

### Mobile application
- React Native with Expo and TypeScript.
- Responsible for UI, navigation, user interaction, and API requests.
- Uses Firebase Authentication for Google Sign-In.
- Does not access Firestore directly.

### Go backend
- Exposes an HTTP REST API.
- Verifies Firebase ID tokens.
- Handles user authorization and business logic.
- Performs all Firestore reads and writes.
- Calculates balances and simplified debt settlements.

### Firebase
- Firebase Authentication: user identity.
- Cloud Firestore: application data.
- Cloud Storage: receipt attachments when implemented.

## Core functionality

- User profiles and Google authentication.
- Groups and group membership.
- Shared expenses with equal and custom splits.
- Expense categories.
- Debt balances and simplification.
- Settlements.
- Friendships and direct expenses.

## Financial invariants

- Monetary values use integer minor units.
- Expense splits must sum to the total expense.
- A user must be authorized to access the relevant group or expense.
- Financial operations must not silently alter historical records.
- Debt simplification produces suggested settlements rather than modifying expenses.

## Backend layout

- `backend/main.go`: application entry point and Firebase/server startup.
- `backend/internal/httpapi/`: HTTP routing, authentication middleware, and request context helpers.
- `backend/internal/database/`: Firebase Admin initialization.
- `backend/tests/httpapi/`: HTTP and middleware tests, kept separate from implementation files and run with `go test ./...`.
- `backend/internal/handlers/`, `models/`, and `services/`: existing empty packages; no application behavior is implemented there yet.

The Go module files remain at the backend root. Local environment configuration also remains there to preserve the existing startup behavior.

## Development status

The Go backend initializes Firebase Admin and serves HTTP on port 8080 using the standard-library router. `GET /api/health` is public. `GET /api/users/me` verifies a Firebase ID token from the Authorization Bearer header and returns the verified UID as JSON. Authentication middleware makes that UID available in the request context.

The Expo Android application uses native Google Sign-In and React Native Firebase Authentication in a development build. Google credentials are exchanged with Firebase; Firebase ID tokens authenticate requests to Go. Native Firebase manages session persistence, and the UI displays the UID returned by Go. The API base URL is configured through `EXPO_PUBLIC_API_BASE_URL`. iOS and web authentication are not configured yet. Firestore user profiles and other application endpoints, database models, and business logic are not yet implemented.

Update this document when architectural decisions are made or changed.
