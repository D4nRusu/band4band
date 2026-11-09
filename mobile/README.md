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

## Groups

The Groups tab shows groups you belong to. Sign in from Home first, then open Groups. Use **Create a new group** to open the name/description dialog. The description is optional. On save, Go creates the group with your verified UID as its first member and the group appears in your list. Pull down to refresh; the list also refreshes when returning to the tab.

Open a group and tap the gear icon in the bottom-right corner of its banner to open **Group settings**. Owners see a red **Delete group** action. Confirming deletes the group for everyone; this cannot be undone. Go checks the verified UID against `createdBy` before deleting, even if someone bypasses the UI. Other members see **Leave group** instead, which removes only their own membership after confirmation.

Names are limited to 1–100 characters and descriptions to 1,000 characters after trimming. Group names need not be unique. Existing memberships are loaded from Go, not directly from Firestore. Invitations are available by exact username through the invite icon beside the members popup title. All current members can invite, not just the owner. Balances, removing other members, and group images remain outside this milestone.

For later device verification: create a group, restart the app, and confirm it remains listed. Sign in with a different account and confirm the first account's group is not visible. Check cancel, empty-name validation, optional descriptions, and backend/network errors in the dialog. A timed-out create may still have reached Go: close the dialog and refresh before submitting again.

Restart Go and reload Metro after updating this code. No APK rebuild is required for these JavaScript and backend changes.

## Group invitations

Open a group, tap **X people**, then tap the invite icon beside **People in [group name]** and enter the person's existing username without `@`. If the group contains only its owner, a hint points to the invite icon. Matching is case-insensitive. Accounts must have chosen their username first. Sending the same group/person invitation again does not create another card or replace its original sender. Existing members cannot be invited again.

The recipient sees invitation cards above their groups, with the group name, inviter, and **Accept / Decline**. Accepting adds the group to their list; declining does not. Another invitation may be sent after a decline. Pull-to-refresh updates both lists. Invitations also refresh every 15 seconds while the Groups tab is active and the app is in the foreground. No system push notifications are sent.

For later device testing, use two or three accounts: invite another user, confirm one card after repeated invitations, accept, and check that the new member can invite a third user. Check decline followed by re-invite, non-owner Leave versus owner Delete, and that deleting a group removes its pending invitations. The owner cannot leave their own group. Invitations sent before a member leaves remain valid until the recipient responds or the owner deletes the group.

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

### Group pages

Tap a group's name or description to open its page within the Groups tab. The banner uses a local placeholder image. Tap **X people** to see member photos, names, and emails; close the popup or use Android Back to return. Pull down on the group page to refresh its details and member count. Member information is fetched through Go when the popup opens and is restricted to current group members.

Restart Go and reload Metro to use these routes; no new APK is needed. Physical-device UI and Firestore runtime verification remain deferred.

### Editing groups

Open a group's gear icon, then **Edit group**. Any current member can change the name and description. Save changes returns to settings and updates the group page. The image button is a disabled placeholder; image uploads are not available yet. Restart Go for the new update endpoint and reload Metro; no APK rebuild is required.

### Group categories

Open **Group settings → Categories** to see the ten shared defaults plus custom categories for that group. Any current member can tap **Add new category**, enter a name of 1–40 characters, and select an icon. Names must differ from existing custom and default names, ignoring case and repeated whitespace. Custom categories are shared with the group's members and do not appear in other groups. Refresh the list to see categories another member added.

Restart Go and reload Metro for the new endpoints and page. No APK rebuild, Firebase Console setup, or dependency installation is needed. Category editing and actual expense creation are not implemented yet. Custom categories have a delete icon on the right; any current member can delete them after confirmation. Defaults cannot be deleted. Runtime/device and emulator tests remain deferred.

### Expenses

Open a group and tap **Add expense**. Enter its name, amount in EUR (for example `12.50` or `12,50`), choose a default/custom category, and select the current member who paid. The payer initially defaults to you; you may create an expense paid by someone else. Save adds it to the group's newest-first Expenses list. Pull down to refresh.

Amounts must be positive with at most two decimal places. Go stores integer cents and derives the expense currency from its group; all groups currently use EUR, including existing groups. Category and payer display information are preserved on the expense if a category is later removed or its payer leaves. Equal splits are implemented; multiple payers, aggregate balances, settlements, and expense editing remain deferred.

Restart Go and reload Metro. No APK rebuild or dependency installation is needed. If saving times out or returns an uncertain result, close the form and refresh expenses before retrying: it may already have saved. Model, HTTP, emulator, and mobile amount-parsing tests are present but execution and physical-device verification remain deferred.

Groups containing expenses cannot be deleted yet. Deletion will require everyone to be settled up once balance/settlement functionality exists; for now the server rejects deletion rather than risking financial history. Empty groups may still be deleted normally.

Tap an expense to open its detail menu, showing its name and total on one row, a smaller category, and who paid with the added date (without time). Saved participant shares appear below. Old expenses without shares still show **No contributions**. Any current member can use the top-right trash icon and confirm deletion. The row disappears after Go confirms, while the original record and deletion audit details remain stored. Groups with expense history still cannot be deleted, even if their visible list is empty. Restart Go and reload Metro; no APK rebuild is required.

### Equal contributions

When adding an expense, all current group members are selected as contributors by default, including the payer. Tap the checkboxes to exclude anyone. The payer and contributors are independent: one member can pay entirely for others, or just for one other person. At least one contributor must remain selected. Go calculates the equal shares, stores them in cents, and returns them for the detail menu. The payer's portion is labeled **Own share**; others are labeled **Owes**.

For leftover cents, Go sorts selected Firebase UIDs and gives one extra cent to the first selected UID(s) until the remainder is exhausted. For example, €10 across usr1, usr2, usr3 becomes €3.34, €3.33, €3.33. Saved shares never change when someone later leaves. Existing expenses without contributions are left untouched. Restart Go and reload Metro together because creating a new expense now requires the participant selection; no APK rebuild is required.
