import { SymbolView } from 'expo-symbols';
import { Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { ListItem } from '@/lib/lists-api';

type ListItemRowProps = {
  item: ListItem;
  currentUID: string;
  memberName: (uid: string) => string;
  disabled: boolean;
  onToggle: (item: ListItem) => void;
  onLongPress: (item: ListItem) => void;
};

export function ListItemRow({ item, currentUID, memberName, disabled, onToggle, onLongPress }: ListItemRowProps) {
  const theme = useTheme();
  const archived = item.clearedAt !== undefined;
  const subtitle = describeItem(item, currentUID, memberName);

  return (
    <Pressable
      onPress={() => onToggle(item)}
      onLongPress={() => onLongPress(item)}
      disabled={disabled || archived}
      accessibilityRole="checkbox"
      accessibilityState={{ checked: item.done, disabled: disabled || archived }}
      accessibilityLabel={`${item.text}${subtitle ? `, ${subtitle}` : ''}`}
      accessibilityHint={archived ? undefined : 'Tap to tick or untick. Long press for more options.'}
      style={({ pressed }) => [
        styles.row,
        { backgroundColor: theme.backgroundElement },
        pressed && styles.pressed,
        archived && styles.archived,
      ]}
    >
      <SymbolView
        name={item.done
          ? { ios: 'checkmark.square', android: 'check_box', web: 'check_box' }
          : { ios: 'square', android: 'check_box_outline_blank', web: 'check_box_outline_blank' }}
        size={24}
        tintColor={theme.text}
      />
      <View style={styles.details}>
        <ThemedText style={item.done && styles.doneText}>{item.text}</ThemedText>
        {subtitle.length > 0 && <ThemedText type="small" themeColor="textSecondary">{subtitle}</ThemedText>}
      </View>
    </Pressable>
  );
}

function describeItem(item: ListItem, currentUID: string, memberName: (uid: string) => string): string {
  const parts: string[] = [];

  if (item.done) {
    const doer = item.doneBy === currentUID ? 'you' : item.doneByName;
    parts.push(item.doneAt ? `Done by ${doer}, ${formatDate(item.doneAt)}` : `Done by ${doer}`);
  } else if (item.assigneeUID) {
    parts.push(item.assigneeUID === currentUID ? 'Your turn' : `Assigned to ${item.assigneeName}`);
  }

  if (item.rotationUIDs.length > 0) {
    parts.push(`Rotation: ${item.rotationUIDs.map((uid) => (uid === currentUID ? 'You' : memberName(uid))).join(' → ')}`);
  }

  return parts.join(' · ');
}

function formatDate(value: string): string {
  const date = new Date(value);
  const month = date.toLocaleString('en-US', { month: 'short' });
  const time = date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });
  return `${month} ${date.getDate()}, ${time}`;
}

const styles = StyleSheet.create({
  row: { flexDirection: 'row', alignItems: 'center', gap: 12, padding: 12, minHeight: 56, borderRadius: 12 },
  details: { flex: 1, minWidth: 0, gap: 2 },
  doneText: { textDecorationLine: 'line-through' },
  pressed: { opacity: 0.7 },
  archived: { opacity: 0.6 },
});
