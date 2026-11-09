import { SymbolView } from 'expo-symbols';
import { Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { GroupMember } from '@/lib/groups-api';

type ExpenseParticipantsProps = {
  members: GroupMember[];
  selectedUIDs: string[];
  disabled: boolean;
  onToggle: (uid: string) => void;
};

export function ExpenseParticipants({ members, selectedUIDs, disabled, onToggle }: ExpenseParticipantsProps) {
  const theme = useTheme();

  return (
    <View style={styles.section}>
      <ThemedText>Split equally between</ThemedText>
      <ThemedText type="small" themeColor="textSecondary">
        Tap to include or exclude anyone, including the payer.
      </ThemedText>

      {members.map((member) => {
        const selected = selectedUIDs.includes(member.uid);
        const name = member.name || member.username || member.uid;

        return (
          <Pressable key={member.uid} onPress={() => onToggle(member.uid)} disabled={disabled}
            accessibilityRole="checkbox" accessibilityLabel={name}
            accessibilityState={{ checked: selected, disabled }}
            style={[styles.row, { backgroundColor: theme.backgroundElement }]}>
            <SymbolView size={24} tintColor={theme.text}
              name={selected
                ? { ios: 'checkmark.square', android: 'check_box', web: 'check_box' }
                : { ios: 'square', android: 'check_box_outline_blank', web: 'check_box_outline_blank' }} />
            <ThemedText style={styles.name}>{name}</ThemedText>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: 8 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 12, padding: 12, minHeight: 48, borderRadius: 12 },
  name: { flex: 1 },
});
