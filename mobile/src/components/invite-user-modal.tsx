import { useEffect, useRef, useState } from 'react';
import { ActivityIndicator, Button, Modal, StyleSheet, TextInput, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import type { Group } from '@/lib/groups-api';
import { sendGroupInvitation } from '@/lib/invitations-api';

type InviteUserModalProps = {
  user: AuthenticatedUser;
  group: Group;
  onClose: () => void;
};

export function InviteUserModal({ user, group, onClose }: InviteUserModalProps) {
  const theme = useTheme();
  const [username, setUsername] = useState('');
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const request = useRef<AbortController | null>(null);

  useEffect(() => {
    return () => {
      request.current?.abort();
      request.current = null;
    };
  }, []);

  async function submit() {
    if (request.current) return;

    const handle = username.trim();
    if (!/^[a-zA-Z0-9_]{3,30}$/.test(handle)) {
      setError('Enter their username without @: 3–30 letters, numbers, or underscores.');
      return;
    }

    const controller = new AbortController();
    request.current = controller;
    setSending(true);
    setError(null);
    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      await sendGroupInvitation(user, group.id, handle, controller.signal);
      if (!controller.signal.aborted) setSent(true);
    } catch (cause) {
      if (request.current !== controller) return;

      let message = 'Could not send this invitation.';
      if (controller.signal.aborted) {
        message = 'The request timed out. You can retry safely; only one invitation will be shown.';
      } else if (cause instanceof Error) {
        message = cause.message;
      }

      setError(message);
    } finally {
      clearTimeout(timeout);
      if (request.current === controller) {
        request.current = null;
        setSending(false);
      }
    }
  }

  function close() {
    if (!sending) onClose();
  }

  return (
    <Modal transparent animationType="fade" onRequestClose={close}>
      <View style={styles.overlay}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <ThemedText type="subtitle">Invite to {group.name}</ThemedText>

          {sent ? (
            <ThemedText>Invitation sent. If one was already pending, the existing invitation was kept.</ThemedText>
          ) : (
            <>
              <TextInput
                accessibilityLabel="Username to invite"
                placeholder="Username (without @)"
                placeholderTextColor={theme.textSecondary}
                autoCapitalize="none"
                autoCorrect={false}
                value={username}
                onChangeText={setUsername}
                editable={!sending}
                style={[styles.input, { color: theme.text, borderColor: theme.textSecondary }]}
              />
              {error && <ThemedText accessibilityRole="alert">{error}</ThemedText>}
              {sending && <ActivityIndicator accessibilityLabel="Sending invitation" />}
              <Button title="Send invitation" disabled={sending} onPress={submit} />
            </>
          )}

          <Button title={sent ? 'Done' : 'Cancel'} disabled={sending} onPress={close} />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: { flex: 1, justifyContent: 'center', padding: 24, backgroundColor: 'rgba(0, 0, 0, 0.5)' },
  dialog: { padding: 24, gap: 16, borderRadius: 16 },
  input: { padding: 12, fontSize: 16, borderWidth: 1, borderRadius: 8 },
});
