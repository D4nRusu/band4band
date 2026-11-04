# Band4band mobile authentication

Android Google Sign-In uses an Expo development build, React Native Firebase, and the existing Go API. Expo Go cannot run these native modules. iOS and web currently show an unsupported-platform message; their authentication configuration is outside this milestone.

## Authentication flow

Google Sign-In returns a Google ID token. `GoogleAuthProvider.credential` and `signInWithCredential` exchange it for a Firebase session. The API helper obtains a **Firebase ID token** using the Firebase user's `getIdToken()` and sends it as `Authorization: Bearer ...` to `GET /api/users/me`. The stored profile returned by Go is displayed as backend confirmation.

Firebase's native SDK restores the session after restarting. No application code writes tokens to files, AsyncStorage, or logs. Sign-out clears the Firebase session and signs out of Google. The SDK refreshes Firebase tokens when needed. No Firestore SDK operations are used.

## One-time Firebase and Expo setup

Use Node 24.12 or newer (the installed dependencies require it); Node 26 was used for verification. From `mobile/`:

```bash
npm ci
npx eas-cli@latest login
npx eas-cli@latest init
```

Link to your intended Expo project or create one. This adds your real Expo project ID; no project ID is supplied in this repository. Review that change. `eas.json` already defines an Android development APK profile.

In Firebase Console:

1. Open the same project used by the Go backend.
2. Under Authentication → Sign-in method, confirm Google is enabled and a project support email is set.
3. Under Project settings → General → Your apps, select Android app `com.band4band.mobile`.
4. Register the **SHA-1 and SHA-256 fingerprints of the certificate signing the actual development APK**. Different build/signing methods can use different keys. Do not assume the upload/release certificate is the development certificate.
5. Download the refreshed `google-services.json` into `mobile/google-services.json`. Its Android package must stay `com.band4band.mobile`.

The Google Sign-In configuration uses the existing Web OAuth client (`client_type: 3`) from that file. It is a public client identifier, not a client secret. If you change Firebase projects or OAuth clients, update `webClientId` in `src/hooks/use-auth-session.android.ts` to the corresponding Web client. Never use an Android client ID for that field.

To manage EAS signing credentials, use:

```bash
npx eas-cli@latest credentials -p android
```

That command can display/manage a keystore, but its fingerprints are relevant only if that key signs this APK. The actual APK certificate is authoritative. If its fingerprints are not known yet, create the first development APK, inspect it, register its certificate in Firebase, download the refreshed configuration, and rebuild before testing sign-in.

Android SDK Build Tools provide this inspection command (replace the path with your downloaded APK):

```bash
apksigner verify --print-certs /path/to/development.apk
```

Copy the printed signer certificate SHA-1 and SHA-256 digests into Firebase. No Android Studio or emulator is needed for EAS cloud builds; Build Tools are needed only for this local APK inspection. If OAuth consent is in Testing mode and restricts users, add your Google account under Google Auth Platform → Audience → Test users in Google Cloud Console.

## Build and install the APK

From `mobile/`, after linking the Expo project:

```bash
npx eas-cli@latest build --platform android --profile development
```

React Native Firebase app/auth are pinned to 26.0.0 (Firebase Android BoM 34.15.0). Version 27 requires Kotlin 2.4, while this Expo/React Native toolchain uses Kotlin 2.1.20. Keep both Firebase packages aligned and use the Expo Kotlin defaults; a Kotlin property override alone did not upgrade the active compiler. After changing native dependencies, rebuild with `npx eas-cli@latest build --platform android --profile development --clear-cache`. Native compilation must still be verified with EAS.

EAS uploads the project and compiles the APK in the cloud. Review the upload: the Firebase **client** configuration must be included, while backend `.env`, service-account keys, and `secrets/` must remain excluded. The repository's ignore rules exclude those backend credentials. If EAS reports that `google-services.json` is absent, check ignore rules and the build archive before retrying.

When the build completes, open the build-details link EAS prints. Either:

- Open that link in the phone's browser, download the APK, allow installation from that browser when Android asks, and install it.
- Download it on Linux and install over USB with `adb install -r /path/to/development.apk`.

Open the installed **mobile** application, not Expo Go. This APK is a development client: it needs Metro running to load the JavaScript application. If Android refuses an update because the installed app has another signing certificate, uninstall the previous `com.band4band.mobile` app first. Uninstalling clears its local Firebase session.

## Run live over USB on Arch Linux

Install Android platform tools if needed:

```bash
sudo pacman -S android-tools
```

On the phone, enable Developer options (usually Settings → About phone → tap Build number seven times), then enable USB debugging. Connect a USB data cable, unlock the phone, and accept its RSA debugging prompt.

```bash
adb devices
```

The phone must appear as `device`, not `unauthorized`. If Linux permissions prevent access, follow Arch's Android device/udev setup for your phone; do not run Metro as root.

In a terminal at the repository root, start Go with its existing Firebase Admin credentials:

```bash
cd backend
go run .
```

In another terminal at the repository root:

```bash
curl http://127.0.0.1:8080/api/health
adb reverse tcp:8080 tcp:8080
adb reverse tcp:8081 tcp:8081
cd mobile
cp .env.example .env.local
npx expo start --dev-client --localhost --port 8081
```

The example sets `EXPO_PUBLIC_API_BASE_URL=http://127.0.0.1:8080`. The first USB mapping forwards phone port 8080 to Go; the second forwards phone port 8081 to Metro. This avoids sending tokens over a Wi-Fi HTTP connection. The phone still needs internet access for Google and Firebase.

Launch the installed development client. Select the Metro server; if it is not discovered, enter `http://127.0.0.1:8081` in the launcher's development-server URL field. You can also press `a` in Expo CLI with the USB-connected phone available. Keep Metro and Go running. Reapply both `adb reverse` commands after reconnecting the phone or restarting adb.

The generated Android **debug** manifest already allows cleartext HTTP for local development. The API helper rejects HTTP outside development; use an HTTPS API endpoint for non-development builds. No release cleartext permission has been added.

`EXPO_PUBLIC_*` values are public application configuration, never secrets. Restart Metro after changing `.env.local`; use `--clear` if an old URL remains. JavaScript changes normally appear through Fast Refresh. Rebuild/reinstall the APK after changing native dependencies, native plugins, or `google-services.json`.

## Physical-device verification

1. Tap Google Sign-In and choose an account. Confirm “Backend authentication confirmed” and the profile returned by Go. For a new profile, choose a username (3–30 letters, digits, or underscores; saved lowercase). A handle already owned by another user produces an inline “already taken” message. After saving, the form is replaced by the saved handle.
2. Close/force-stop the app, reopen the same Metro project, and confirm it restores the session and checks Go without a new sign-in.
3. Tap Sign out, restart again, and confirm the Google button returns.
4. Cancel account selection; the app should return to idle without an error.
5. Confirm the Google photo (or default avatar), name, email, username, and joined date are displayed. Sign out and back in: the saved handle and joined date must not change. Use another account to try the same username with different capitalization; it must be rejected.
6. Stop Go and tap Check backend again. Confirm a network error or timeout, restart Go, and retry.
7. If Google reports a configuration error, verify the APK signing fingerprints, package name, Web client ID, and Firebase project before rebuilding.

A Firebase sign-in alone is not backend confirmation. A backend 401 generally means the Firebase token/project does not match the Go verification configuration. Check that mobile and backend use the same Firebase project.

## Automated checks

```bash
npm test
npx tsc --noEmit
npm run lint
npx expo config --type public
```

Tests live in `tests/` and use Node's built-in test runner with native TypeScript stripping, without a new test framework. They check the Firebase token Authorization header, endpoint, configuration validation, production HTTP rejection, 401 handling, invalid backend responses, profile submission, and username-conflict feedback. A harmless Node module-type detection warning can appear when importing the Expo TypeScript helper directly.

Android prebuild can be checked with `npx expo prebuild --platform android --no-install`; it generates ignored native directories. Do not edit those directories by hand. Full cross-platform introspection requires iOS Firebase configuration, which is not supplied here.

References: [Expo Firebase](https://docs.expo.dev/guides/using-firebase/), [Google Sign-In Expo setup](https://react-native-google-signin.github.io/docs/setting-up/expo), [APK builds and installation](https://docs.expo.dev/build-reference/apk/), [development client usage](https://docs.expo.dev/develop/development-builds/use-development-builds/).
