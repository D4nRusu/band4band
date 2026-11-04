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
- `backend/internal/database/`: Firebase Admin initialization and Firestore user/profile storage.
- `backend/internal/models/`: profile fields, verified identity data, and username validation.
- `backend/tests/`: HTTP, model, and Firestore emulator tests, separate from implementation files.
- `backend/internal/handlers/` and `services/`: existing empty packages.

The Go module files remain at the backend root. Local environment configuration also remains there to preserve the existing startup behavior.

## Development status

The Go backend initializes Firebase Admin and serves HTTP on port 8080 using the standard-library router. `GET /api/health` is public. `GET /api/users/me` verifies a Firebase ID token from the Authorization Bearer header and returns the stored user profile as JSON. Authentication middleware makes the UID and verified identity claims available in the request context.

The Expo Android application uses native Google Sign-In and React Native Firebase Authentication in a development build. Google credentials are exchanged with Firebase; Firebase ID tokens authenticate requests to Go. Native Firebase manages session persistence, and the UI displays the profile returned by Go, including Google photo (or a default avatar), name, email, username, creation date, and UID. First-login onboarding asks for a unique username. The API base URL is configured through `EXPO_PUBLIC_API_BASE_URL`. iOS and web authentication are not configured yet. Friendships, groups, expenses, photo uploads, profile editing, and user-search endpoints are not yet implemented.

## User profiles and onboarding

On the application's first authenticated `GET /api/users/me` (also called when restoring a session), Go creates `users/{firebaseUID}` if missing. Creation uses Firestore `Create`, so concurrent first requests cannot overwrite another request's document. A failed or delayed mobile request can be retried. This is application-driven provisioning, not a Firebase Authentication trigger; an Auth account alone does not create a Firestore document until Go receives a profile request.

Profile fields are `uid`, `email`, `name`, `username`, `photoURL`, `createdAt`, and `profileComplete`. Initial UID/email/name/photo come exclusively from verified Firebase token claims. `createdAt` is a Firestore server timestamp and is returned as an RFC 3339 JSON string. New users have an empty username and `profileComplete: false`. Subsequent logins read the existing profile without resynchronizing or overwriting these fields from Google.

Authenticated `PUT /api/users/me/username` accepts only `{"username":"chosen_handle"}`. Usernames contain 3–30 ASCII letters, digits, or underscores and are stored in lowercase. A Firestore transaction creates `usernames/{normalizedUsername}` with the owning UID and updates the user's username and `profileComplete` together. This makes uniqueness case-insensitive and prevents partial reservations and concurrent duplicate claims. The reservation collection provides a future exact-handle lookup without exposing emails; no friend search API is implemented in this milestone.

A taken username returns HTTP 409 with code `username_taken`, shown inline in the mobile form. Invalid usernames return HTTP 400. Completed profiles cannot choose a different username in this milestone; submitting the same normalized handle again succeeds for retry safety. Profile writes never accept client-provided UID, email, photo, timestamps, or completion flags. Both profile endpoints require Firebase authentication; health remains public. Profile responses use `Cache-Control: no-store`.

All Firestore operations use Go's existing Admin client. Mobile uses Firebase Authentication only and does not read or write Firestore. Firestore client security rules should continue denying direct client access; Admin writes use backend IAM permissions.

## Profile verification

Unit/HTTP checks run with `go test ./...`; Firestore integration tests in `backend/tests/database` skip unless `FIRESTORE_EMULATOR_HOST` is set. They use isolated `demo-profile-*` project IDs and never intentionally connect to a production database. They cover concurrent profile creation, preservation on subsequent logins, retry safety, and racing case-insensitive username claims. Run them against an already-started local emulator with `FIRESTORE_EMULATOR_HOST=127.0.0.1:8085 go test ./tests/database -v` from `backend/`.

Update this document when architectural decisions are made or changed.
