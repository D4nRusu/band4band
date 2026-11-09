import { ActivityIndicator, Button, StyleSheet, View } from 'react-native';

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

  return (
    <View style={styles.section}>
      <ThemedText type="subtitle" accessibilityRole="header">Expenses</ThemedText>
      <Button title="Add expense" onPress={onAdd} disabled={loading || saving || error !== null} />
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
  section: { padding: 24, gap: 16 },
  feedback: { gap: 8 },
  card: { padding: 16, gap: 8, borderRadius: 12 },
});
