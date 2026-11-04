import {
  getAuth, GoogleAuthProvider, onAuthStateChanged, signInWithCredential, signOut,
  type User,
} from '@react-native-firebase/auth';
import {
  GoogleSignin, isErrorWithCode, isSuccessResponse, statusCodes,
} from '@react-native-google-signin/google-signin';
import { useEffect, useState } from 'react';

// Web OAuth client (client_type 3) from this app's google-services.json.
const webClientId = '170534880306-blln60bru62nijeo7qakjgl9qujtmfsj.apps.googleusercontent.com';

export function useAuthSession() {
  const [user, setUser] = useState<User | null>(null);
  const [initializing, setInitializing] = useState(true);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    GoogleSignin.configure({ webClientId });

    return onAuthStateChanged(getAuth(), (currentUser) => {
      setUser(currentUser);
      setError(null);
      setInitializing(false);
    });
  }, []);

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
      const message = signInErrorMessage(cause, stage);
      if (message) setError(message);
    } finally {
      setWorking(false);
    }
  }

  async function logout() {
    setWorking(true);
    setError(null);
    try {
      await signOut(getAuth());
      await GoogleSignin.signOut();
    } catch {
      setError('Sign-out failed. Please try again.');
    } finally {
      setWorking(false);
    }
  }

  return { user, initializing, working, error, login, logout };
}

function signInErrorMessage(cause: unknown, stage: string): string | null {
  if (isErrorWithCode(cause) && cause.code === statusCodes.SIGN_IN_CANCELLED) return null;

  // SDK error objects can contain credentials. Only expose recognized error codes.
  let code = 'unknown';
  if (isErrorWithCode(cause)) {
    const isNativeCode = /^\d{1,5}$/.test(cause.code);
    const isFirebaseCode = /^auth\/[a-z-]{1,60}$/.test(cause.code);
    const isGoogleCode = Object.values(statusCodes).includes(cause.code);
    if (isNativeCode || isFirebaseCode || isGoogleCode) code = cause.code;
  }

  const configurationMismatch = code === '10' || code === 'DEVELOPER_ERROR';
  const hint = configurationMismatch
    ? 'The APK signing certificate, Android package, or OAuth client does not match Google configuration.'
    : 'Report this stage and error code to diagnose the failure.';

  return `${stage} failed (${code}). ${hint}`;
}
