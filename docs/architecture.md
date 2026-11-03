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

## Development status

The Go backend initializes Firebase Admin and serves HTTP on port 8080 using the standard-library router. `GET /api/health` is public. `GET /api/users/me` verifies a Firebase ID token from the Authorization Bearer header and returns the verified UID as JSON. Authentication middleware makes that UID available in the request context.

The starter Expo application is established. Firestore user profiles and other application endpoints, database models, and business logic are not yet implemented.

Update this document when architectural decisions are made or changed.
