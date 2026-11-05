import { ActivityIndicator, Button, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { GroupInvitation } from '@/lib/invitations-api';

type GroupInvitationsProps = {
  invitations: GroupInvitation[];
  loading: boolean;
  error: string | null;
  respondingID: string | null;
  disabled: boolean;
  onRespond: (id: string, accept: boolean) => Promise<void>;
};

export function GroupInvitations({ invitations, loading, error, respondingID, disabled, onRespond }: GroupInvitationsProps) {
  const theme = useTheme();

  return (
    <View style={styles.section}>
      {loading && <ActivityIndicator accessibilityLabel="Loading invitations" />}
      {error && <ThemedText accessibilityRole="alert">{error}</ThemedText>}
      {invitations.length > 0 && <ThemedText type="subtitle">Invitations</ThemedText>}

      {invitations.map((invitation) => {
        const inviter = invitation.inviterUsername
          ? `@${invitation.inviterUsername}`
          : invitation.inviterName || 'A group member';
        const responding = respondingID === invitation.id;

        return (
          <View key={invitation.id} style={[styles.card, { backgroundColor: theme.backgroundElement }]}>
            <ThemedText type="smallBold">{invitation.groupName}</ThemedText>
            <ThemedText>{inviter} invited you to join this group.</ThemedText>
            {responding && <ActivityIndicator accessibilityLabel="Responding to invitation" />}

            <View style={styles.actions}>
              <Button title="Accept" disabled={disabled || respondingID !== null}
                onPress={() => { void onRespond(invitation.id, true); }} />
              <Button title="Decline" disabled={disabled || respondingID !== null}
                onPress={() => { void onRespond(invitation.id, false); }} />
            </View>
          </View>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: 12 },
  card: { padding: 20, borderRadius: 12, gap: 12 },
  actions: { flexDirection: 'row', gap: 16 },
});
