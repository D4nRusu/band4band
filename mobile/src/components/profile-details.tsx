import { useState } from 'react';
import { Image, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { UserProfile } from '@/lib/api';

export function ProfileDetails({ profile }: { profile: UserProfile }) {
  const theme = useTheme();
  const [photoFailed, setPhotoFailed] = useState(false);
  const showPhoto = profile.photoURL.startsWith('https://') && !photoFailed;
  const displayName = profile.name || profile.username || 'Your profile';
  const initial = (profile.name || profile.username || '?').slice(0, 1).toUpperCase();

  return (
    <>
      {showPhoto ? (
        <Image
          source={{ uri: profile.photoURL }}
          style={styles.avatar}
          accessibilityLabel="Google profile picture"
          onError={() => setPhotoFailed(true)}
        />
      ) : (
        <View
          style={[styles.avatar, styles.defaultAvatar, { backgroundColor: theme.backgroundElement }]}
          accessible
          accessibilityLabel="Default profile avatar">
          <ThemedText type="subtitle">{initial}</ThemedText>
        </View>
      )}

      <ThemedText type="subtitle">{displayName}</ThemedText>
      <ThemedText selectable>{profile.email || 'No email available'}</ThemedText>
      {profile.profileComplete && <ThemedText selectable>@{profile.username}</ThemedText>}
      <ThemedText type="small">Joined {new Date(profile.createdAt).toLocaleDateString()}</ThemedText>
      <ThemedText selectable type="small">UID: {profile.uid}</ThemedText>
    </>
  );
}

const styles = StyleSheet.create({
  avatar: { width: 80, height: 80, borderRadius: 40 },
  defaultAvatar: { justifyContent: 'center', alignItems: 'center' },
});
