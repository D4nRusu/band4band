import { useState } from 'react';
import { Image, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { GroupMember } from '@/lib/groups-api';

export function GroupMemberRow({ member }: { member: GroupMember }) {
  const theme = useTheme();
  const [photoFailed, setPhotoFailed] = useState(false);
  const name = member.name || member.username || 'Member';
  const showPhoto = member.photoURL.startsWith('https://') && !photoFailed;

  return (
    <View style={styles.row}>
      {showPhoto ? (
        <Image source={{ uri: member.photoURL }} style={styles.avatar}
          accessibilityLabel={`${name}'s profile picture`} onError={() => setPhotoFailed(true)} />
      ) : (
        <View style={[styles.avatar, styles.fallback, { backgroundColor: theme.backgroundElement }]}
          accessible accessibilityLabel={`${name}'s default avatar`}>
          <ThemedText>{Array.from(name)[0].toUpperCase()}</ThemedText>
        </View>
      )}

      <View style={styles.details}>
        <ThemedText type="smallBold">{name}</ThemedText>
        <ThemedText selectable type="small">{member.email || 'No email available'}</ThemedText>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 16, paddingVertical: 12 },
  avatar: { width: 48, height: 48, borderRadius: 24 },
  fallback: { alignItems: 'center', justifyContent: 'center' },
  details: { flex: 1, gap: 4 },
});
