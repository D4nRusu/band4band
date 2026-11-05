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

The Expo Android application uses native Google Sign-In and React Native Firebase Authentication in a development build. Google credentials are exchanged with Firebase; Firebase ID tokens authenticate requests to Go. Native Firebase manages session persistence, and the UI displays the profile returned by Go, including Google photo (or a default avatar), name, email, username, creation date, and UID. First-login onboarding asks for a unique username. The API base URL is configured through `EXPO_PUBLIC_API_BASE_URL`. iOS and web authentication are not configured yet. Basic group creation and membership-filtered listing are implemented. Group invitations, acceptance/decline, and non-owner leaving are implemented. Friendships, removing other members, expenses, photo uploads, profile editing, and general user-search endpoints are not yet implemented.

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

Groups are stored at `groups/{autoGeneratedID}` with `name`, `description`, `createdBy`, `memberUIDs`, a server-generated `createdAt`, and an internal `pendingInvitations` map when invitations are present. The API also returns the document ID as `id`. Creation writes the group and `memberUIDs: [creatorUID]` in a single document operation, so the group cannot be created without its initial membership. Membership is stored as an array on the group. The creator is its owner; no additional named roles are implemented. Both endpoints require authentication; no additional completed-profile prerequisite is imposed in this milestone. Membership is added through accepted invitations. No endpoint exists for removing another member. Current members may edit the group name and description.

The mobile Groups tab is an Expo Router stack under `src/app/groups/`, with the list at `index.tsx` and group details at `[groupID].tsx`. Android uses `groups-screen.android.tsx`, `use-groups.ts`, and `groups-api.ts`. It restores the Firebase session, lists the current user's groups, refreshes on focus or pull-to-refresh, and shows empty/loading/error states. A dedicated `create-group-modal.tsx` collects the name and optional description. Successful creation updates the list and closes the modal. List requests are cancelled when the tab loses focus. Pending creation is cancelled when the screen unmounts, including on sign-out or account changes. Creation is never automatically retried: if it times out, the user is asked to refresh the list before retrying because the server may have committed it. No balances or expenses are calculated or displayed yet.

As with profiles, the mobile app makes only authenticated HTTP calls to Go for groups; all Firestore operations remain in Go. No native dependencies were added, so a Metro reload is sufficient alongside restarting Go.

### Owner-only group deletion

The existing `createdBy` field defines the group owner. Authenticated `DELETE /api/groups/{groupID}` reads the group and verifies `createdBy == authenticatedUID` in a Firestore transaction before deleting it. Non-owners receive HTTP 403, missing groups return 404, and successful deletion returns 204 with no body. The client cannot supply an owner UID. Since membership is embedded in the group, deletion removes the current memberships with that document. The same transaction deletes the pending invitation documents referenced by the group. No expenses or financial history exist yet; a future expenses milestone must preserve that history when defining deletion behavior.

The group banner has a settings icon in its bottom-right corner. Group settings shows a red Delete group action for the owner or Leave group for another member, followed by explicit confirmation. `use-exit-group.ts` handles the request, cancellation, loading, and errors. A successful action dismisses to the Groups list, which refreshes on focus. Failed or timed-out requests keep settings open with feedback; closing settings and refreshing can reconcile an uncertain result. A 404 on retry is treated as already removed by the client. Owners must delete rather than leave.

## Invitations and leaving groups

Any current member can `POST /api/groups/{groupID}/invitations` with `{"username":"alice"}`. Go normalizes the exact username, resolves the existing `usernames/{username}` reservation, and verifies that the recipient has a completed profile. It checks the sender's current membership before resolving users, and rejects invitations to anyone already in the group, including the sender. No general username-search or email lookup endpoint is exposed.

Pending invitations are stored in `groupInvitations/{autoID}` with `groupID`, `groupName`, `recipientUID`, `inviterUID`, `inviterUsername`, `inviterName`, and a server `createdAt`. The inviter's username comes from their saved Go profile; the verified Firebase name is a display fallback. The group's internal `pendingInvitations` map records `recipientUID -> invitationID`, using explicit Firestore field paths to handle arbitrary UID characters. It is not returned in group JSON. Existing groups need no migration: the map is added on the first invitation.

Invitation creation reads membership and pending state in a transaction, then writes both the invitation and pending map entry atomically. Repeated or concurrent invitations to the same group/person preserve the first pending invitation and its sender. A recipient can still have invitations to multiple different groups.

`GET /api/invitations` queries only the authenticated recipient's pending documents, filters against the current group/pending map/membership, and returns cards newest first. Deleted groups and stale invitations are not displayed. Querying by recipient and sorting in Go avoids adding a composite index.

Only the recipient may `POST /api/invitations/{invitationID}/accept` or `/decline`. A transaction checks the recipient and the exact pending invitation ID. Accepting adds the UID with Firestore `ArrayUnion`; declining leaves membership unchanged. Both responses delete the pending document and map entry. No invitation history is retained. A later re-invite gets a new random ID, so a delayed response to an older invitation cannot accept or decline the replacement. Resolved/missing invitations return 404, unauthorized responses return 403, and successful actions return 204.

Authenticated `POST /api/groups/{groupID}/leave` removes only the verified user's UID with `ArrayRemove` in a transaction. A repeat leave by a non-owner is a safe no-op. The owner receives 403 and must use deletion instead. Invitations issued by a member remain valid if that member later leaves; ownership remains with the creator. These operations do not alter financial history.

The Groups list header renders invitation cards containing the group name, inviter, Accept, and Decline. Group cards open their detail page and have no invite or destructive controls. The people modal has an invite icon beside its title, opening the existing username invitation dialog. When only the owner is present, a hint points to that icon. Closing the invite dialog returns to the member list. Owners see Delete and other members see Leave in Group settings, with confirmation. Accepting refreshes the membership-filtered group list. Pull-to-refresh loads groups and invitations together. The inbox polls Go every 15 seconds only while Groups is focused and the app is active, refreshes on returning to the foreground, and cancels list requests on blur/background. There are no OS push notifications or direct mobile Firestore calls.

HTTP authorization/validation and emulator lifecycle/concurrency tests are present in `backend/tests/httpapi/invitations_test.go` and `backend/tests/database/invitations_test.go`. Execution is deferred at the user's request.

## Profile verification

Unit/HTTP checks run with `go test ./...`; Firestore integration tests in `backend/tests/database` skip unless `FIRESTORE_EMULATOR_HOST` is set. They use isolated `demo-profile-*` project IDs and never intentionally connect to a production database. They cover concurrent profile creation, preservation on subsequent logins, retry safety, and racing case-insensitive username claims. Run them against an already-started local emulator with `FIRESTORE_EMULATOR_HOST=127.0.0.1:8085 go test ./tests/database -v` from `backend/`.

Update this document when architectural decisions are made or changed.

## Group detail pages

Selecting a group opens its detail page within the Groups tab. A local placeholder image with a dark overlay displays the group name and current membership count. The people button opens a scrollable modal with member photos (or default avatars), names, and emails. No balances are implemented.

Authenticated `GET /api/groups/{groupID}` returns the group only to current members. `GET /api/groups/{groupID}/members` returns member UID, name, email, username, and photo URL from saved user profiles. Go checks membership and reads profiles in a Firestore transaction, so unauthorized callers cannot retrieve member emails. Missing profiles retain a member row with fallback display text. Neither endpoint writes data or accepts a client-selected caller UID. Invalid IDs return 400, non-members 403, and missing groups 404.

The mobile detail screen uses `use-group-details.ts` for focus/pull refresh and `use-group-members.ts` for loading members when the popup opens. Requests have cancellation and timeout feedback. All data continues to pass through Go; no mobile Firestore dependency or native build change is required.

Successful leaving or deletion returns navigation to the Groups list. A retained group detail route also dismisses to the list when Go returns 403 (membership lost) or 404 (group deleted), including after signing in again. Connection and server errors retain the retry UI.

## Editing group details

Group settings includes Edit group for every current member. The edit modal initializes the existing name and description, validates the same limits as creation, and saves through authenticated `PUT /api/groups/{groupID}`. Only name and description are accepted. Go checks current membership and updates those two fields in one Firestore transaction, preserving ownership, memberships, timestamps, and pending invitations. Concurrent edits use the last committed name/description. Invitation cards already resolve the current group name when loaded.

The saved group updates the open page immediately; the Groups list refreshes on focus. `use-edit-group.ts` handles save loading, cancellation, timeouts, and errors. The edit menu has an explicitly disabled Change image (coming later) button. Image uploads and storage configuration remain unimplemented; no dependencies were added. HTTP validation/authorization and Firestore member-edit tests were added, with execution deferred.
