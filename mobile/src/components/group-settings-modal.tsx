import { ActivityIndicator, Alert, Button, Modal, Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useExitGroup } from '@/hooks/use-exit-group';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import type { Group } from '@/lib/groups-api';

type GroupSettingsModalProps = {
  user: AuthenticatedUser & { uid: string };
  group: Group;
  onClose: () => void;
  onEdit: () => void;
};

export function GroupSettingsModal({ user, group, onClose, onEdit }: GroupSettingsModalProps) {
  const theme = useTheme();
  const isOwner = group.createdBy === user.uid;
  const { working, error, exitGroup } = useExitGroup(user, group, isOwner);
  const actionLabel = isOwner ? 'Delete group' : 'Leave group';

  function close() {
    if (!working) onClose();
  }

  function confirmExit() {
    const title = isOwner ? 'Delete group?' : 'Leave group?';
    const message = isOwner
      ? `Delete “${group.name}” for everyone in the group? This cannot be undone.`
      : `Leave “${group.name}”? The group will remain available to its other members.`;

    Alert.alert(title, message, [
      { text: 'Cancel', style: 'cancel' },
      { text: actionLabel, style: 'destructive', onPress: () => void exitGroup() },
    ]);
  }

  return (
    <Modal transparent animationType="fade" onRequestClose={close}>
      <View style={styles.overlay}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <ThemedText type="subtitle" accessibilityRole="header">Group settings</ThemedText>

          <Button title="Edit group" disabled={working} onPress={onEdit} />

          <Pressable onPress={confirmExit} disabled={working} accessibilityRole="button"
            accessibilityLabel={actionLabel} accessibilityState={{ disabled: working }} style={styles.action}>
            <ThemedText style={styles.destructive}>{actionLabel}</ThemedText>
          </Pressable>

          {working && <ActivityIndicator accessibilityLabel={isOwner ? 'Deleting group' : 'Leaving group'} />}
          {error && <ThemedText accessibilityRole="alert">{error}</ThemedText>}

          <Button title="Close" disabled={working} onPress={close} />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: { flex: 1, justifyContent: 'center', padding: 24, backgroundColor: 'rgba(0, 0, 0, 0.5)' },
  dialog: { padding: 24, gap: 16, borderRadius: 16 },
  action: { minHeight: 48, justifyContent: 'center' },
  destructive: { color: '#d32f2f' },
});
