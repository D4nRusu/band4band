import { StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import { categoryIcons } from '@/lib/categories-api';
import type { Expense } from '@/lib/expenses-api';
import { formatExpenseAmount } from '@/lib/money';

export function ExpenseCard({ expense }: { expense: Expense }) {
  const theme = useTheme();
  const date = new Date(expense.createdAt);
  const month = date.toLocaleString('en-US', { month: 'short' });
  const dateLabel = `${month}. ${date.getDate()}`;

  return (
    <View style={[styles.card, { backgroundColor: theme.backgroundElement }]}>
      <ThemedText type="small" themeColor="textSecondary" style={styles.date}>{dateLabel}</ThemedText>
      <ThemedText style={styles.icon} accessibilityLabel={expense.categoryName}>
        {categoryIcons[expense.categoryIcon].image}
      </ThemedText>

      <View style={styles.details}>
        <ThemedText style={styles.title} numberOfLines={1}>{expense.name}</ThemedText>
        <ThemedText type="small" themeColor="textSecondary" numberOfLines={1}>Paid by {expense.payerName}</ThemedText>
      </View>

      <ThemedText style={styles.amount} numberOfLines={1} adjustsFontSizeToFit>
        {formatExpenseAmount(expense.amountMinor, expense.currency)}
      </ThemedText>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flexDirection: 'row', alignItems: 'center', padding: 12, gap: 8, minHeight: 72, borderRadius: 12 },
  date: { width: 50 },
  icon: { width: 32, fontSize: 28, lineHeight: 36, textAlign: 'center' },
  details: { flex: 1, minWidth: 0, gap: 2 },
  title: { fontSize: 18, lineHeight: 24, fontWeight: '600' },
  amount: { maxWidth: '35%', fontSize: 16, lineHeight: 24, fontWeight: '600', textAlign: 'right' },
});
