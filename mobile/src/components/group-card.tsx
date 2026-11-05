import { Pressable, StyleSheet } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { Group } from '@/lib/groups-api';

type GroupCardProps = {
  group: Group;
  disabled: boolean;
  onOpen: (group: Group) => void;
};

export function GroupCard({ group, disabled, onOpen }: GroupCardProps) {
  const theme = useTheme();

  return (
    <Pressable
      onPress={() => onOpen(group)}
      disabled={disabled}
      accessibilityRole="link"
      accessibilityLabel={`Open group ${group.name}`}
      style={[styles.card, { backgroundColor: theme.backgroundElement }]}>
      <ThemedText type="subtitle">{group.name}</ThemedText>
      {group.description !== '' && <ThemedText>{group.description}</ThemedText>}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: { padding: 20, borderRadius: 12, gap: 8 },
});
