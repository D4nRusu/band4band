import { Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';

type ExpenseChoicesProps = {
  label: string;
  choices: { id: string; label: string }[];
  selected: string;
  disabled: boolean;
  onSelect: (id: string) => void;
};

export function ExpenseChoices({ label, choices, selected, disabled, onSelect }: ExpenseChoicesProps) {
  const theme = useTheme();

  return (
    <View style={styles.section}>
      <ThemedText>{label}</ThemedText>
      <View style={styles.choices}>
        {choices.map((choice) => (
          <Pressable key={choice.id} onPress={() => onSelect(choice.id)} disabled={disabled}
            accessibilityRole="radio" accessibilityLabel={choice.label}
            accessibilityState={{ checked: selected === choice.id, disabled }}
            style={[styles.choice, { backgroundColor: theme.backgroundElement,
              borderColor: selected === choice.id ? theme.text : 'transparent' }]}>
            <ThemedText type="small">{choice.label}</ThemedText>
          </Pressable>
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: 8 },
  choices: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  choice: { padding: 12, minHeight: 48, justifyContent: 'center', borderRadius: 12, borderWidth: 2 },
});
