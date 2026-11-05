import { ActivityIndicator, Button, FlatList, Modal, StyleSheet, View } from 'react-native';

import { GroupMemberRow } from '@/components/group-member-row';
import { ThemedText } from '@/components/themed-text';
import { useGroupMembers } from '@/hooks/use-group-members';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import type { Group } from '@/lib/groups-api';

type GroupMembersModalProps = {
  user: AuthenticatedUser;
  group: Group;
  onClose: () => void;
};

export function GroupMembersModal({ user, group, onClose }: GroupMembersModalProps) {
  const theme = useTheme();
  const { members, loading, error, retry } = useGroupMembers(user, group.id);

  return (
    <Modal transparent animationType="fade" onRequestClose={onClose}>
      <View style={styles.overlay}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <ThemedText type="subtitle" accessibilityRole="header">People in {group.name}</ThemedText>
          {loading && <ActivityIndicator accessibilityLabel="Loading group members" />}

          {error && (
            <View style={styles.feedback}>
              <ThemedText accessibilityRole="alert">{error}</ThemedText>
              <Button title="Retry" disabled={loading} onPress={retry} />
            </View>
          )}

          <FlatList
            data={members}
            keyExtractor={(member) => member.uid}
            renderItem={({ item }) => <GroupMemberRow member={item} />}
            style={styles.list}
          />

          <Button title="Close" onPress={onClose} />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: { flex: 1, justifyContent: 'center', padding: 24, backgroundColor: 'rgba(0, 0, 0, 0.5)' },
  dialog: { width: '100%', height: '75%', maxWidth: 600, alignSelf: 'center', borderRadius: 16, padding: 24, gap: 16 },
  feedback: { gap: 12 },
  list: { flex: 1 },
});
