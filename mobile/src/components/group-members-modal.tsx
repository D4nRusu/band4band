import { SymbolView } from 'expo-symbols';
import { ActivityIndicator, Button, FlatList, Modal, Pressable, StyleSheet, View } from 'react-native';

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
  onInvite: () => void;
};

export function GroupMembersModal({ user, group, onClose, onInvite }: GroupMembersModalProps) {
  const theme = useTheme();
  const { members, loading, error, retry } = useGroupMembers(user, group.id);

  return (
    <Modal transparent animationType="fade" onRequestClose={onClose}>
      <View style={styles.overlay}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <View style={styles.header}>
            <ThemedText style={styles.title} type="subtitle" accessibilityRole="header">People in {group.name}</ThemedText>
            <Pressable onPress={onInvite} accessibilityRole="button"
              accessibilityLabel="Invite people" style={styles.invite}>
              <SymbolView name={{ ios: 'person.badge.plus', android: 'person_add', web: 'person_add' }}
                tintColor={theme.text} size={24} />
            </Pressable>
          </View>

          {!loading && !error && members.length === 1 && members[0].uid === group.createdBy && (
            <ThemedText>Just you so far. Tap the invite icon above to invite more people.</ThemedText>
          )}
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
  header: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  title: { flex: 1 },
  invite: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center' },
  feedback: { gap: 12 },
  list: { flex: 1 },
});
