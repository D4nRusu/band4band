import { Alert, Button, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { Group } from '@/lib/groups-api';

type GroupCardProps = {
  group: Group;
  currentUID: string;
  disabled: boolean;
  deleting: boolean;
  onDelete: (groupID: string) => Promise<void>;
};

export function GroupCard({ group, currentUID, disabled, deleting, onDelete }: GroupCardProps) {
  const theme = useTheme();
  const isOwner = group.createdBy === currentUID;

  function confirmDelete() {
    Alert.alert(
      'Delete group?',
      `Delete “${group.name}” for everyone in the group? This cannot be undone.`,
      [
        { text: 'Cancel', style: 'cancel' },
        { text: 'Delete', style: 'destructive', onPress: () => { void onDelete(group.id); } },
      ],
    );
  }

  return (
    <View style={[styles.card, { backgroundColor: theme.backgroundElement }]}>
      <ThemedText type="subtitle">{group.name}</ThemedText>
      {group.description !== '' && <ThemedText>{group.description}</ThemedText>}

      {isOwner && (
        <Button
          title={deleting ? 'Deleting…' : 'Delete group'}
          accessibilityLabel={`Delete group ${group.name}`}
          disabled={disabled}
          onPress={confirmDelete}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  card: { padding: 20, borderRadius: 12, gap: 8 },
});
