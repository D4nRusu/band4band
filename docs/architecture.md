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
- `backend/internal/httpapi/`: HTTP routing, authentication middleware, and request context helpers. Route registration is in `routes.go`, profile handlers in `users.go`, and response/error mapping in `user_responses.go`.
- `backend/internal/database/`: Firebase Admin initialization and Firestore profile/group storage.
- `backend/internal/models/`: profile/group fields and input validation.
- `backend/tests/`: HTTP, model, and Firestore emulator tests, separate from implementation files.
- `backend/internal/handlers/` and `services/`: existing empty packages.

The Go module files remain at the backend root. Local environment configuration also remains there to preserve the existing startup behavior.

## Development status

The Go backend initializes Firebase Admin and serves HTTP on port 8080 using the standard-library router. `GET /api/health` is public. `GET /api/users/me` verifies a Firebase ID token from the Authorization Bearer header and returns the stored user profile as JSON. Authentication middleware makes the UID and verified identity claims available in the request context.

The Expo Android application uses native Google Sign-In and React Native Firebase Authentication in a development build. Google credentials are exchanged with Firebase; Firebase ID tokens authenticate requests to Go. Native Firebase manages session persistence, and the UI displays the profile returned by Go, including Google photo (or a default avatar), name, email, username, creation date, and UID. First-login onboarding asks for a unique username. The API base URL is configured through `EXPO_PUBLIC_API_BASE_URL`. iOS and web authentication are not configured yet. Basic group creation and membership-filtered listing are implemented. Friendships, invitations, adding/removing members, expenses, photo uploads, profile editing, and user-search endpoints are not yet implemented.

## Mobile authentication layout

- `src/components/auth-screen.android.tsx`: screen composition and authenticated profile controls.
- `src/hooks/use-auth-session.android.ts`: Google/Firebase sign-in, session restoration, and sign-out.
- `src/hooks/use-user-profile.ts`: profile loading, username submission, request cancellation, and feedback state.
- `src/components/profile-details.tsx` and `username-form.tsx`: profile display and username form rendering.
- `src/lib/api.ts`: authenticated profile requests and response validation.

The authenticated profile subtree is keyed by Firebase UID, so profile and form state are discarded on sign-out or account changes.

## User profiles and onboarding

On the application's first authenticated `GET /api/users/me` (also called when restoring a session), Go creates `users/{firebaseUID}` if missing. Creation uses Firestore `Create`, so concurrent first requests cannot overwrite another request's document. A failed or delayed mobile request can be retried. This is application-driven provisioning, not a Firebase Authentication trigger; an Auth account alone does not create a Firestore document until Go receives a profile request.

Profile fields are `uid`, `email`, `name`, `username`, `photoURL`, `createdAt`, and `profileComplete`. Initial UID/email/name/photo come exclusively from verified Firebase token claims. `createdAt` is a Firestore server timestamp and is returned as an RFC 3339 JSON string. New users have an empty username and `profileComplete: false`. Subsequent logins read the existing profile without resynchronizing or overwriting these fields from Google.

Authenticated `PUT /api/users/me/username` accepts only `{"username":"chosen_handle"}`. Usernames contain 3–30 ASCII letters, digits, or underscores and are stored in lowercase. A Firestore transaction creates `usernames/{normalizedUsername}` with the owning UID and updates the user's username and `profileComplete` together. This makes uniqueness case-insensitive and prevents partial reservations and concurrent duplicate claims. The reservation collection provides a future exact-handle lookup without exposing emails; no friend search API is implemented in this milestone.

A taken username returns HTTP 409 with code `username_taken`, shown inline in the mobile form. Invalid usernames return HTTP 400. Completed profiles cannot choose a different username in this milestone; submitting the same normalized handle again succeeds for retry safety. Profile writes never accept client-provided UID, email, photo, timestamps, or completion flags. Both profile endpoints require Firebase authentication; health remains public. Profile responses use `Cache-Control: no-store`.

All Firestore operations use Go's existing Admin client. Mobile uses Firebase Authentication only and does not read or write Firestore. Firestore client security rules should continue denying direct client access; Admin writes use backend IAM permissions.

## Groups and initial membership

Authenticated `GET /api/groups` returns a JSON array of groups whose `memberUIDs` contains the verified Firebase UID. The client cannot choose another user's UID. The query uses Firestore's single-field array membership index; results are sorted newest first in Go, avoiding a composite index requirement.

Authenticated `POST /api/groups` accepts only `{"name":"Weekend trip","description":"Shared travel costs"}`. Names are required and limited to 1–100 Unicode code points after trimming; descriptions are optional and limited to 1,000 code points after trimming. Duplicate group names are allowed. Client-provided IDs, creator UIDs, member UIDs, or timestamps are rejected.

Groups are stored at `groups/{autoGeneratedID}` with `name`, `description`, `createdBy`, `memberUIDs`, and a server-generated `createdAt`. The API also returns the document ID as `id`. Creation writes the group and `memberUIDs: [creatorUID]` in a single document operation, so the group cannot be created without its initial membership. Membership is currently stored as an array on the group, with no additional role or permission model. Both endpoints require authentication; no additional completed-profile prerequisite is imposed in this milestone. No endpoint exists yet for inviting, adding/removing members, or editing/deleting groups.

The mobile Groups tab is a separate Expo Router page at `src/app/groups.tsx`. Android uses `groups-screen.android.tsx`, `use-groups.ts`, and `groups-api.ts`. It restores the Firebase session, lists the current user's groups, refreshes on focus or pull-to-refresh, and shows empty/loading/error states. A dedicated `create-group-modal.tsx` collects the name and optional description. Successful creation updates the list and closes the modal. List requests are cancelled when the tab loses focus. Pending creation is cancelled when the screen unmounts, including on sign-out or account changes. Creation is never automatically retried: if it times out, the user is asked to refresh the list before retrying because the server may have committed it. No balances or expenses are calculated or displayed yet.

As with profiles, the mobile app makes only authenticated HTTP calls to Go for groups; all Firestore operations remain in Go. No native dependencies were added, so a Metro reload is sufficient alongside restarting Go.

## Profile verification

Unit/HTTP checks run with `go test ./...`; Firestore integration tests in `backend/tests/database` skip unless `FIRESTORE_EMULATOR_HOST` is set. They use isolated `demo-profile-*` project IDs and never intentionally connect to a production database. They cover concurrent profile creation, preservation on subsequent logins, retry safety, and racing case-insensitive username claims. Run them against an already-started local emulator with `FIRESTORE_EMULATOR_HOST=127.0.0.1:8085 go test ./tests/database -v` from `backend/`.

Update this document when architectural decisions are made or changed.
