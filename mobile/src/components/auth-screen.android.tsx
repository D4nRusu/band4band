import {
  getAuth, GoogleAuthProvider, onAuthStateChanged, signInWithCredential, signOut,
  type User,
} from '@react-native-firebase/auth';
import {
  GoogleSignin, GoogleSigninButton, isErrorWithCode, isSuccessResponse, statusCodes,
} from '@react-native-google-signin/google-signin';
import { useEffect, useState } from 'react';
import { ActivityIndicator, Button, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ThemedText } from '@/components/themed-text';
import { getCurrentUser } from '@/lib/api';

// Web OAuth client (client_type 3) from this app's google-services.json.
const webClientId = '170534880306-blln60bru62nijeo7qakjgl9qujtmfsj.apps.googleusercontent.com';

export default function AuthScreen() {
  const [user, setUser] = useState<User | null>(null);
  const [initializing, setInitializing] = useState(true);
  const [working, setWorking] = useState(false);
  const [checking, setChecking] = useState(false);
  const [uid, setUID] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    GoogleSignin.configure({ webClientId });
    return onAuthStateChanged(getAuth(), (currentUser) => {
      setUser(currentUser);
      setUID(null);
      setError(null);
      setChecking(currentUser !== null);
      setInitializing(false);
    });
  }, []);

  useEffect(() => {
    if (!user) return;
    const controller = new AbortController();
    const timeout = setTimeout(() => {
      controller.abort();
      setChecking(false);
      setError('The backend request timed out. Check the API URL and connection, then retry.');
    }, 15000);
    getCurrentUser(user, controller.signal)
      .then((verifiedUID) => {
        if (!controller.signal.aborted) setUID(verifiedUID);
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) {
          setError(cause instanceof Error ? cause.message : 'Could not reach the backend.');
        }
      })
      .finally(() => {
        clearTimeout(timeout);
        if (!controller.signal.aborted) setChecking(false);
      });
    return () => {
      clearTimeout(timeout);
      controller.abort();
    };
  }, [user, attempt]);

  async function login() {
    setWorking(true);
    setError(null);
    let stage = 'Google Play services check';
    try {
      await GoogleSignin.hasPlayServices({ showPlayServicesUpdateDialog: true });
      stage = 'Google account sign-in';
      const result = await GoogleSignin.signIn();
      if (!isSuccessResponse(result)) return;
      if (!result.data.idToken) {
        setError('Google returned no ID token. Check the Web OAuth client configuration.');
        return;
      }
      const credential = GoogleAuthProvider.credential(result.data.idToken);
      stage = 'Firebase credential exchange';
      await signInWithCredential(getAuth(), credential);
    } catch (cause) {
      if (isErrorWithCode(cause) && cause.code === statusCodes.SIGN_IN_CANCELLED) return;
      // Show only a recognized error code; SDK error objects can contain credentials.
      const code = isErrorWithCode(cause) && (
        /^\d{1,5}$/.test(cause.code)
        || /^auth\/[a-z-]{1,60}$/.test(cause.code)
        || Object.values(statusCodes).includes(cause.code)
      ) ? cause.code : 'unknown';
      const hint = code === '10' || code === 'DEVELOPER_ERROR'
        ? 'The APK signing certificate, Android package, or OAuth client does not match Google configuration.'
        : 'Report this stage and error code to diagnose the failure.';
      setError(`${stage} failed (${code}). ${hint}`);
    } finally {
      setWorking(false);
    }
  }

  async function logout() {
    setWorking(true);
    setError(null);
    try {
      await signOut(getAuth());
      setChecking(false);
      await GoogleSignin.signOut();
    } catch {
      setError('Sign-out failed. Please try again.');
    } finally {
      setWorking(false);
    }
  }

  const busy = initializing || working || checking;
  return (
    <SafeAreaView style={styles.container}>
      <ThemedText type="title">Band4band</ThemedText>
      {busy && <ActivityIndicator accessibilityLabel="Loading authentication" />}
      {!initializing && !user && (
        <GoogleSigninButton disabled={busy} onPress={login} style={styles.googleButton} />
      )}
      {user && (
        <>
          <ThemedText>{uid ? 'Backend authentication confirmed' : 'Signed in to Firebase'}</ThemedText>
          {uid && <ThemedText selectable>UID: {uid}</ThemedText>}
          <Button title="Check backend again" disabled={busy} onPress={() => {
            setChecking(true);
            setError(null);
            setUID(null);
            setAttempt(attempt + 1);
          }} />
          <Button title="Sign out" disabled={working} onPress={logout} />
        </>
      )}
      {error && <ThemedText accessibilityRole="alert">{error}</ThemedText>}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, justifyContent: 'center', alignItems: 'center', padding: 24, gap: 20 },
  googleButton: { width: 220, height: 48 },
});
