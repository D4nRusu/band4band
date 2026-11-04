import { GoogleSigninButton } from '@react-native-google-signin/google-signin';
import { ActivityIndicator, Button, ScrollView, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ProfileDetails } from '@/components/profile-details';
import { ThemedText } from '@/components/themed-text';
import { UsernameForm } from '@/components/username-form';
import { useAuthSession } from '@/hooks/use-auth-session.android';
import { useTheme } from '@/hooks/use-theme';
import { useUserProfile } from '@/hooks/use-user-profile';
import type { AuthenticatedUser } from '@/lib/api';

export default function AuthScreen() {
  const theme = useTheme();
  const { user, initializing, working, error, login, logout } = useAuthSession();

  return (
    <SafeAreaView style={[styles.safeArea, { backgroundColor: theme.background }]}>
      <ScrollView contentContainerStyle={styles.container} keyboardShouldPersistTaps="handled">
        <ThemedText type="title">Band4band</ThemedText>

        {!user && (initializing || working) && (
          <ActivityIndicator accessibilityLabel="Loading authentication" />
        )}

        {!initializing && !user && (
          <GoogleSigninButton disabled={working} onPress={login} style={styles.googleButton} />
        )}

        {user && (
          <AuthenticatedProfile key={user.uid} user={user} authWorking={working} onSignOut={logout} />
        )}

        {error && <ThemedText accessibilityRole="alert">{error}</ThemedText>}
      </ScrollView>
    </SafeAreaView>
  );
}

type AuthenticatedProfileProps = {
  user: AuthenticatedUser;
  authWorking: boolean;
  onSignOut: () => Promise<void>;
};

function AuthenticatedProfile({ user, authWorking, onSignOut }: AuthenticatedProfileProps) {
  const {
    profile, checking, saving, error,
    username, usernameError, changeUsername, saveUsername, reload,
  } = useUserProfile(user);
  const busy = authWorking || checking || saving;

  return (
    <>
      {busy && <ActivityIndicator accessibilityLabel="Loading authentication" />}
      <ThemedText>{profile ? 'Backend authentication confirmed' : 'Signed in to Firebase'}</ThemedText>

      {profile && <ProfileDetails profile={profile} />}

      {profile && !profile.profileComplete && (
        <UsernameForm
          username={username}
          error={usernameError}
          disabled={busy}
          onChange={changeUsername}
          onSubmit={saveUsername}
        />
      )}

      <Button title="Check backend again" disabled={busy} onPress={reload} />
      <Button title="Sign out" disabled={authWorking || saving} onPress={onSignOut} />
      {error && <ThemedText accessibilityRole="alert">{error}</ThemedText>}
    </>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1 },
  container: { flexGrow: 1, justifyContent: 'center', alignItems: 'center', padding: 24, gap: 20 },
  googleButton: { width: 220, height: 48 },
});
