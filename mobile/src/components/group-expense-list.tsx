import { ActivityIndicator, Button, Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import { categoryIcons } from '@/lib/categories-api';
import type { Expense } from '@/lib/expenses-api';
import { formatExpenseAmount } from '@/lib/money';

type GroupExpenseListProps = {
  expenses: Expense[];
  loading: boolean;
  error: string | null;
  saving: boolean;
  onAdd: () => void;
  onRetry: () => void;
};

export function GroupExpenseList({ expenses, loading, error, saving, onAdd, onRetry }: GroupExpenseListProps) {
  const theme = useTheme();
  const addDisabled = loading || saving || error !== null;

  return (
    <View style={styles.section}>
      <Pressable onPress={onAdd} disabled={addDisabled} accessibilityRole="button"
        accessibilityLabel="Add expense" accessibilityState={{ disabled: addDisabled }}
        style={({ pressed }) => [styles.add, { backgroundColor: theme.backgroundElement },
          pressed && styles.pressed, addDisabled && styles.disabled]}>
        <ThemedText style={styles.plus} accessible={false}>+</ThemedText>
        <ThemedText type="smallBold">Add expense</ThemedText>
      </Pressable>
      {loading && <ActivityIndicator accessibilityLabel="Loading expenses" />}
      {error && (
        <View style={styles.feedback}>
          <ThemedText accessibilityRole="alert">{error}</ThemedText>
          <Button title="Retry expenses" onPress={onRetry} disabled={loading || saving} />
        </View>
      )}
      {!loading && !error && expenses.length === 0 && <ThemedText>No expenses yet. Add the first one above.</ThemedText>}

      {expenses.map((expense) => (
        <View key={expense.id} style={[styles.card, { backgroundColor: theme.backgroundElement }]}>
          <ThemedText type="smallBold">{expense.name}</ThemedText>
          <ThemedText>{formatExpenseAmount(expense.amountMinor, expense.currency)}</ThemedText>
          <ThemedText type="small">{categoryIcons[expense.categoryIcon].image} {expense.categoryName}</ThemedText>
          <ThemedText type="small">Paid by {expense.payerName}</ThemedText>
          <ThemedText type="small" themeColor="textSecondary">{new Date(expense.createdAt).toLocaleString()}</ThemedText>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { paddingHorizontal: 24, paddingTop: 12, paddingBottom: 24, gap: 16 },
  add: { alignSelf: 'flex-start', flexDirection: 'row', alignItems: 'center', gap: 8,
    minHeight: 48, paddingHorizontal: 14, borderRadius: 24 },
  plus: { fontSize: 24, lineHeight: 28 },
  pressed: { opacity: 0.7 },
  disabled: { opacity: 0.5 },
  feedback: { gap: 8 },
  card: { padding: 16, gap: 8, borderRadius: 12 },
});
