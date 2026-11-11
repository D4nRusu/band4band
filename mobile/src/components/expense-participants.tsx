import { SymbolView } from 'expo-symbols';
import { Pressable, StyleSheet, TextInput, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { GroupMember } from '@/lib/groups-api';

type ExpenseParticipantsProps = {
  members: GroupMember[];
  selectedUIDs: string[];
  disabled: boolean;
  onToggle: (uid: string) => void;
  mode: 'equal' | 'unequal';
  amounts: Record<string, string>;
  onChangeAmount: (uid: string, value: string) => void;
};

export function ExpenseParticipants({ members, selectedUIDs, disabled, onToggle, mode, amounts, onChangeAmount }: ExpenseParticipantsProps) {
  const theme = useTheme();

  return (
    <View style={styles.section}>
      <ThemedText>{mode === 'equal' ? 'Split equally between' : 'Contribution amounts (€)'}</ThemedText>
      <ThemedText type="small" themeColor="textSecondary">
        Tap to include or exclude anyone, including the payer.
      </ThemedText>

      {members.map((member) => {
        const selected = selectedUIDs.includes(member.uid);
        const name = member.name || member.username || member.uid;

        return (
          <View key={member.uid} style={[styles.row, { backgroundColor: theme.backgroundElement }]}>
            <Pressable onPress={() => onToggle(member.uid)} disabled={disabled}
              accessibilityRole="checkbox" accessibilityLabel={name}
              accessibilityState={{ checked: selected, disabled }} style={styles.selection}>
              <SymbolView size={24} tintColor={theme.text}
                name={selected
                  ? { ios: 'checkmark.square', android: 'check_box', web: 'check_box' }
                  : { ios: 'square', android: 'check_box_outline_blank', web: 'check_box_outline_blank' }} />
              <ThemedText style={styles.name}>{name}</ThemedText>
            </Pressable>
            {mode === 'unequal' && selected && (
              <TextInput value={amounts[member.uid] ?? ''}
                onChangeText={(value) => onChangeAmount(member.uid, value)} editable={!disabled}
                keyboardType="decimal-pad" accessibilityLabel={`Contribution in euros for ${name}`}
                placeholder="0.00" placeholderTextColor={theme.textSecondary}
                style={[styles.amount, { color: theme.text, borderColor: theme.textSecondary }]} />
            )}
          </View>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: 8 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 12, padding: 12, minHeight: 48, borderRadius: 12 },
  selection: { flex: 1, flexDirection: 'row', alignItems: 'center', gap: 12, minHeight: 48 },
  amount: { width: 96, padding: 8, borderWidth: 1, borderRadius: 8, fontSize: 16, textAlign: 'right' },
  name: { flex: 1 },
});
