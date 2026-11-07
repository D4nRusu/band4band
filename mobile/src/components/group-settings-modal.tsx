import { SymbolView } from 'expo-symbols';
import { ActivityIndicator, Alert, Modal, Pressable, StyleSheet, View } from 'react-native';

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
          <View style={styles.header}>
            <Pressable onPress={close} disabled={working} accessibilityRole="button"
              accessibilityLabel="Back to group" accessibilityState={{ disabled: working }}
              style={({ pressed }) => [styles.back, pressed && styles.pressed, working && styles.disabled]}>
              <SymbolView name={{ ios: 'arrow.left', android: 'arrow_back', web: 'arrow_back' }}
                tintColor={theme.text} size={24} />
            </Pressable>
            <ThemedText style={styles.title} type="subtitle" accessibilityRole="header">Group settings</ThemedText>
          </View>

          <Pressable onPress={onEdit} disabled={working} accessibilityRole="button"
            accessibilityLabel="Edit group" accessibilityState={{ disabled: working }}
            style={({ pressed }) => [styles.action, { backgroundColor: theme.backgroundElement },
              pressed && styles.pressed, working && styles.disabled]}>
            <SymbolView name={{ ios: 'pencil', android: 'edit', web: 'edit' }} tintColor={theme.text} size={24} />
            <ThemedText style={styles.actionLabel}>Edit group</ThemedText>
          </Pressable>

          <Pressable onPress={confirmExit} disabled={working} accessibilityRole="button"
            accessibilityLabel={actionLabel} accessibilityState={{ disabled: working }}
            style={({ pressed }) => [styles.action, { backgroundColor: theme.backgroundElement },
              pressed && styles.pressed, working && styles.disabled]}>
            <SymbolView
              name={isOwner
                ? { ios: 'trash', android: 'delete', web: 'delete' }
                : { ios: 'rectangle.portrait.and.arrow.right', android: 'logout', web: 'logout' }}
              tintColor="#d32f2f" size={24} />
            <ThemedText style={[styles.actionLabel, styles.destructive]}>{actionLabel}</ThemedText>
          </Pressable>

          {working && <ActivityIndicator accessibilityLabel={isOwner ? 'Deleting group' : 'Leaving group'} />}
          {error && <ThemedText accessibilityRole="alert">{error}</ThemedText>}

        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: { flex: 1, justifyContent: 'center', padding: 24, backgroundColor: 'rgba(0, 0, 0, 0.5)' },
  dialog: { width: '100%', maxWidth: 500, alignSelf: 'center', padding: 20, gap: 12, borderRadius: 20 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 8, marginBottom: 8 },
  back: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center', borderRadius: 24 },
  title: { flex: 1, fontSize: 24, lineHeight: 32 },
  action: { minHeight: 60, flexDirection: 'row', alignItems: 'center', padding: 16, gap: 16, borderRadius: 12 },
  actionLabel: { flex: 1, fontWeight: '600' },
  pressed: { opacity: 0.7 },
  disabled: { opacity: 0.5 },
  destructive: { color: '#d32f2f' },
});
