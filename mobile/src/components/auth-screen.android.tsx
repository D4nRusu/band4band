import {
  getAuth, GoogleAuthProvider, onAuthStateChanged, signInWithCredential, signOut,
  type User,
} from '@react-native-firebase/auth';
import {
  GoogleSignin, GoogleSigninButton, isErrorWithCode, isSuccessResponse, statusCodes,
} from '@react-native-google-signin/google-signin';
import { useEffect, useRef, useState } from 'react';
import { ActivityIndicator, Button, Image, ScrollView, StyleSheet, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import { completeProfile, getCurrentUser, type UserProfile } from '@/lib/api';

// Web OAuth client (client_type 3) from this app's google-services.json.
const webClientId = '170534880306-blln60bru62nijeo7qakjgl9qujtmfsj.apps.googleusercontent.com';

export default function AuthScreen() {
  const [user, setUser] = useState<User | null>(null);
  const [initializing, setInitializing] = useState(true);
  const [working, setWorking] = useState(false);
  const [checking, setChecking] = useState(false);
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [username, setUsername] = useState('');
  const [usernameError, setUsernameError] = useState<string | null>(null);
  const [photoFailed, setPhotoFailed] = useState(false);
  const saveRequest = useRef<AbortController | null>(null);
  const theme = useTheme();
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    GoogleSignin.configure({ webClientId });
    const unsubscribe = onAuthStateChanged(getAuth(), (currentUser) => {
      saveRequest.current?.abort();
      setUser(currentUser);
      setProfile(null);
      setUsername('');
      setUsernameError(null);
      setPhotoFailed(false);
      setError(null);
      setChecking(currentUser !== null);
      setInitializing(false);
    });
    return () => {
      unsubscribe();
      saveRequest.current?.abort();
    };
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
      .then((currentProfile) => {
        if (!controller.signal.aborted) setProfile(currentProfile);
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

  async function saveUsername() {
    if (!user) return;
    if (!/^[a-zA-Z0-9_]{3,30}$/.test(username)) {
      setUsernameError('Use 3–30 letters, numbers, or underscores.');
      return;
    }
    setWorking(true);
    setUsernameError(null);
    const controller = new AbortController();
    saveRequest.current = controller;
    const timeout = setTimeout(() => controller.abort(), 15000);
    try {
      const saved = await completeProfile(user, username, controller.signal);
      if (!controller.signal.aborted) setProfile(saved);
    } catch (cause) {
      if (saveRequest.current === controller) {
        setUsernameError(controller.signal.aborted
          ? 'Saving timed out. Retry or reload your profile to check whether it was saved.'
          : cause instanceof Error ? cause.message : 'Could not save your username. Please retry.');
      }
    } finally {
      clearTimeout(timeout);
      if (saveRequest.current === controller) {
        saveRequest.current = null;
        setWorking(false);
      }
    }
  }

  const busy = initializing || working || checking;
  return (
    <SafeAreaView style={[styles.safeArea, { backgroundColor: theme.background }]}>
      <ScrollView contentContainerStyle={styles.container} keyboardShouldPersistTaps="handled">
        <ThemedText type="title">Band4band</ThemedText>
        {busy && <ActivityIndicator accessibilityLabel="Loading authentication" />}
        {!initializing && !user && (
          <GoogleSigninButton disabled={busy} onPress={login} style={styles.googleButton} />
        )}
        {user && (
          <>
            <ThemedText>{profile ? 'Backend authentication confirmed' : 'Signed in to Firebase'}</ThemedText>
            {profile && (
              <>
                {profile.photoURL.startsWith('https://') && !photoFailed ? (
                  <Image source={{ uri: profile.photoURL }} style={styles.avatar}
                    accessibilityLabel="Google profile picture" onError={() => setPhotoFailed(true)} />
                ) : (
                  <View style={[styles.avatar, styles.defaultAvatar, { backgroundColor: theme.backgroundElement }]}
                    accessible accessibilityLabel="Default profile avatar">
                    <ThemedText type="subtitle">{(profile.name || profile.username || '?').slice(0, 1).toUpperCase()}</ThemedText>
                  </View>
                )}
                <ThemedText type="subtitle">{profile.name || profile.username || 'Your profile'}</ThemedText>
                <ThemedText selectable>{profile.email || 'No email available'}</ThemedText>
                {profile.profileComplete ? (
                  <ThemedText selectable>@{profile.username}</ThemedText>
                ) : (
                  <>
                    <ThemedText>Choose a unique username so friends can find you.</ThemedText>
                    <TextInput accessibilityLabel="Username" value={username}
                      onChangeText={(value) => { setUsername(value); setUsernameError(null); }}
                      autoCapitalize="none" autoCorrect={false} maxLength={30}
                      editable={!busy} placeholder="Username" placeholderTextColor={theme.textSecondary}
                      style={[styles.input, { color: theme.text, borderColor: theme.textSecondary }]}
                      onSubmitEditing={() => { if (!busy) void saveUsername(); }} />
                    <ThemedText type="small">3–30 letters, numbers, or underscores. Saved in lowercase.</ThemedText>
                    {usernameError && <ThemedText accessibilityRole="alert">{usernameError}</ThemedText>}
                    <Button title="Save username" disabled={busy} onPress={saveUsername} />
                  </>
                )}
                <ThemedText type="small">Joined {new Date(profile.createdAt).toLocaleDateString()}</ThemedText>
                <ThemedText selectable type="small">UID: {profile.uid}</ThemedText>
              </>
            )}
            <Button title="Check backend again" disabled={busy} onPress={() => {
              setChecking(true);
              setError(null);
              setProfile(null);
              setAttempt(attempt + 1);
            }} />
            <Button title="Sign out" disabled={working} onPress={logout} />
          </>
        )}
        {error && <ThemedText accessibilityRole="alert">{error}</ThemedText>}
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1 },
  container: { flexGrow: 1, justifyContent: 'center', alignItems: 'center', padding: 24, gap: 20 },
  avatar: { width: 80, height: 80, borderRadius: 40 },
  defaultAvatar: { justifyContent: 'center', alignItems: 'center' },
  input: { width: '100%', borderWidth: 1, borderRadius: 8, padding: 12, fontSize: 16 },
  googleButton: { width: 220, height: 48 },
});
