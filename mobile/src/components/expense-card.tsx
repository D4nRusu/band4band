import { Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import { categoryIcons } from '@/lib/categories-api';
import type { Expense } from '@/lib/expenses-api';
import { formatExpenseAmount } from '@/lib/money';

type ExpenseCardProps = {
  expense: Expense;
  onOpen: (expense: Expense) => void;
  disabled: boolean;
};

export function ExpenseCard({ expense, onOpen, disabled }: ExpenseCardProps) {
  const theme = useTheme();
  const date = new Date(expense.createdAt);
  const month = date.toLocaleString('en-US', { month: 'short' });

  return (
    <Pressable onPress={() => onOpen(expense)} disabled={disabled} accessibilityRole="button"
      accessibilityLabel={`View expense ${expense.name}`} accessibilityState={{ disabled }}
      style={({ pressed }) => [styles.card, { backgroundColor: theme.backgroundElement }, pressed && styles.pressed]}>
      <View style={styles.date}>
        <ThemedText type="small" themeColor="textSecondary">{month}.</ThemedText>
        <ThemedText type="small" themeColor="textSecondary">{date.getDate()}</ThemedText>
      </View>
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
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: { flexDirection: 'row', alignItems: 'center', padding: 12, gap: 8, minHeight: 72, borderRadius: 12 },
  pressed: { opacity: 0.7 },
  date: { width: 36, alignItems: 'center' },
  icon: { width: 32, fontSize: 28, lineHeight: 36, textAlign: 'center' },
  details: { flex: 1, minWidth: 0, gap: 2 },
  title: { fontSize: 18, lineHeight: 24, fontWeight: '600' },
  amount: { maxWidth: '35%', fontSize: 16, lineHeight: 24, fontWeight: '600', textAlign: 'right' },
});
