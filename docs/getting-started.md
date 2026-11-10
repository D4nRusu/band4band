# Collaborator quick start

Run the Go API on your computer and the Expo development app on a physical Android phone. **Use the team's existing Firebase and Expo projects.** Android is the supported authentication platform; Expo Go cannot run this app's native sign-in modules.

## 1. Get access and files

Ask a maintainer for:

- Repository access and the Firebase development project to use.
- An approved backend credential setup. The instructions below use a service-account JSON key stored **outside the repository**; ask for a credential with appropriate development Firestore access, not an unrestricted personal copy of someone's key.
- The latest Android **development APK** compatible with this checkout. Reusing it avoids a cloud build.
- `mobile/google-services.json`, if missing. This is Firebase's Android client configuration, **not** an Admin credential.
- Expo team access if you need to build an APK yourself. The app is already linked to `odobasians-team`; do not run `eas init` or replace the existing project ID.

The backend credentials and Android configuration must target the **same Firebase project**. Google sign-in must be enabled and the default Firestore database must exist. A maintainer should confirm these are already set up. If Google OAuth testing restricts accounts, have your Google account added as a test user.

Never put Admin credentials in `mobile/`, an `EXPO_PUBLIC_*` variable, commits, or build uploads. Local setup against the team's cloud Firestore reads and writes that project's real data; use the agreed development project.

## 2. Install local tools

- Git, npm, and Node.js. Node 26 is used by the current development environment.
- Go **1.27.2 or newer**, matching `backend/go.mod`.
- An Android phone with Google Play services, internet access, and a USB data cable.
- Android platform tools (`adb`). On Arch Linux:

```bash
sudo pacman -S --needed android-tools
```

Clone the team repository and open its root. Install the locked mobile dependencies:

```bash
cd mobile
npm ci
cp -n .env.example .env.local
```

Confirm `mobile/.env.local` contains:

```dotenv
EXPO_PUBLIC_API_BASE_URL=http://127.0.0.1:8080
```

This localhost URL works on the phone through the USB forwarding configured below. This variable is public configuration, not a secret.

## 3. Start the backend

In a terminal at the repository root, replace the credential path below with your actual approved JSON key path:

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/absolute/path/to/your-approved-key.json"
cd backend
go run .
```

The existing Firebase Admin initialization discovers these Application Default Credentials. No Firebase CLI login is required for this method. Keep the terminal open; Go listens on port **8080**.

In another terminal, confirm the API is reachable:

```bash
curl --fail http://127.0.0.1:8080/api/health
```

Expect a successful JSON response. This checks the public route; signing in and loading your profile will also check protected API and Firestore access.

## 4. Connect the phone and install the app

On Android, enable Developer options (tap Build number seven times), then **USB debugging**. Connect the phone, unlock it, and accept the computer's debugging prompt.

```bash
adb devices
```

Your phone should show as `device`, not `unauthorized`. If Linux reports a permission problem, configure the appropriate Android/udev device permissions; do not run the development servers as root.

Install the maintainer's APK by opening its download link on the phone, or from your computer:

```bash
adb install -r /path/to/development.apk
```

Replace the APK path with the downloaded file. Open the installed **mobile** app, not Expo Go. If Android rejects an update because the signing key differs, ask which APK to use; uninstalling the old app clears its saved session.

## 5. Start live development

Forward the backend and Metro ports:

```bash
adb reverse tcp:8080 tcp:8080
adb reverse tcp:8081 tcp:8081
```

In a second terminal at the repository root:

```bash
cd mobile
npx expo start --dev-client --localhost --port 8081
```

Open the installed app and select this development server. If discovery fails, enter `http://127.0.0.1:8081` in its server URL field, or press `a` in the Expo terminal.

**Keep Go and Metro running.** Reapply both `adb reverse` commands after reconnecting the phone. The phone still needs internet for Google/Firebase; the API and Metro traffic use USB.

## 6. Confirm everything works

1. Sign in with Google. On first login, choose a unique username.
2. Confirm your profile loads from Go, including name, email, and photo/default avatar.
3. Create a test group and expense. Check the category, payer, and selected equal-split contributors.
4. Restart the app and confirm your session is restored.
5. Sign out and confirm the sign-in screen returns.

Expenses and group membership are stored through Go. The mobile app talks directly to Firebase only for authentication; it sends a Firebase ID token to Go for application data requests. Never log those tokens.

## If you need to build an APK

From `mobile/`, with access to the existing Expo team:

```bash
npx eas-cli@latest login
npx eas-cli@latest build --platform android --profile development
```

Download the resulting APK and install it as above. Make sure `google-services.json` is included in the build; Admin keys must never be included. A development APK still needs Metro running. See [Expo's APK guide](https://docs.expo.dev/build-reference/apk/) and [development build usage](https://docs.expo.dev/develop/development-builds/use-development-builds/).

Google sign-in requires the **actual APK signing certificate's SHA-1 and SHA-256** registered under Firebase → Project settings → Android app `com.band4band.mobile`. Reusing the same signed APK needs no new fingerprint for each collaborator. For a differently signed APK, inspect it using Android SDK Build Tools:

```bash
apksigner verify --print-certs /path/to/development.apk
```

Register those certificate digests, download the updated `google-services.json`, then rebuild/reinstall if configuration changed. **The Expo build “Fingerprint” is a project compatibility hash, not a signing-certificate digest.** See [Expo Fingerprint](https://docs.expo.dev/versions/v57.0.0/sdk/fingerprint/) and [Firebase Google sign-in setup](https://firebase.google.com/docs/auth/android/google-signin).

The configured `webClientId` in `mobile/src/hooks/use-auth-session.android.ts` must match the Firebase project's **Web OAuth client**, not its Android client. Don't change it just because you joined the project.

## Daily changes and common problems

| Situation | What to do |
|---|---|
| Changed TypeScript/UI | Fast Refresh normally applies it; reload the app if needed. |
| Changed Go code | Restart `go run .`. |
| Changed `.env.local` | Restart Metro; add `--clear` if stale settings persist. |
| Changed native dependencies, plugins, or Firebase client configuration | Build and install a compatible new APK. |
| App can't reach Go or Metro | Check both terminals, `adb devices`, both reverse mappings, and `/api/health`. |
| Google sign-in error `10` | Check actual APK certificate digests, Android package, and Web OAuth client. Internet alone won't fix this. |
| Backend returns 401 after Google login | Check that Android and Go target the same Firebase project. |
| Backend credential/permission error | Check the key path and approved IAM access; don't weaken Firestore rules. Admin access uses IAM. |

For code orientation, read [architecture.md](architecture.md). Longer Android setup notes are in [mobile/README.md](../mobile/README.md). For the Admin credential mechanism, see [Firebase Admin setup](https://firebase.google.com/docs/admin/setup).

Optional checks, from the respective directories:

```bash
# backend/
go test ./...
go vet ./...

# mobile/
npm test
npx tsc --noEmit
npm run lint
```

Firestore integration tests require a separately configured local emulator; see the architecture document. Full frontend checks currently have known unrelated failures in `group-card.tsx` (`type="body"`) and `use-color-scheme.web.ts` (state update in an effect). Neither check replaces testing on the phone.
