import { Alert, Button, Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { Group } from '@/lib/groups-api';

type GroupCardProps = {
  group: Group;
  currentUID: string;
  disabled: boolean;
  removing: boolean;
  onDelete: (groupID: string) => Promise<void>;
  onLeave: (groupID: string) => Promise<void>;
  onInvite: (group: Group) => void;
  onOpen: (group: Group) => void;
};

export function GroupCard({ group, currentUID, disabled, removing, onDelete, onLeave, onInvite, onOpen }: GroupCardProps) {
  const theme = useTheme();
  const isOwner = group.createdBy === currentUID;

  function confirmExit() {
    const title = isOwner ? 'Delete group?' : 'Leave group?';
    const message = isOwner
      ? `Delete “${group.name}” for everyone in the group? This cannot be undone.`
      : `Leave “${group.name}”? The group will remain available to its other members.`;

    function confirm() {
      if (isOwner) void onDelete(group.id);
      else void onLeave(group.id);
    }

    Alert.alert(title, message, [
      { text: 'Cancel', style: 'cancel' },
      { text: isOwner ? 'Delete' : 'Leave', style: 'destructive', onPress: confirm },
    ]);
  }

  let actionLabel = isOwner ? 'Delete group' : 'Leave group';
  if (removing) actionLabel = isOwner ? 'Deleting…' : 'Leaving…';

  return (
    <View style={[styles.card, { backgroundColor: theme.backgroundElement }]}>
      <Pressable onPress={() => onOpen(group)} disabled={disabled} accessibilityRole="link"
        accessibilityLabel={`Open group ${group.name}`} style={styles.summary}>
        <ThemedText type="subtitle">{group.name}</ThemedText>
        {group.description !== '' && <ThemedText>{group.description}</ThemedText>}
      </Pressable>

      <Button title="Invite user" disabled={disabled} onPress={() => onInvite(group)} />
      <Button title={actionLabel} disabled={disabled} onPress={confirmExit} />
    </View>
  );
}

const styles = StyleSheet.create({
  summary: { gap: 8, paddingVertical: 8 },
  card: { padding: 20, borderRadius: 12, gap: 8 },
});
