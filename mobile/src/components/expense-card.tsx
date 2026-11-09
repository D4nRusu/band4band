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
  const position = expense.viewerPosition;
  const hasMoney = position.kind === 'lent' || position.kind === 'borrowed';
  const moneyColor = position.kind === 'lent' ? theme.moneyReceivable : theme.moneyOwed;

  let subtitle = `Paid by ${expense.payerName}`;
  if (position.kind === 'not_involved') subtitle = 'You are not part of this expense';
  if (position.kind === 'unallocated') subtitle = 'Contributions not set';

  const positionLabel = position.kind === 'lent' ? 'You lent' : 'You borrowed';
  const accessibleMoney = hasMoney ? `${positionLabel} ${formatExpenseAmount(position.amountMinor, expense.currency)}` : '';

  return (
    <Pressable onPress={() => onOpen(expense)} disabled={disabled} accessibilityRole="button"
      accessibilityLabel={`View expense ${expense.name}, ${expense.categoryName}, ${month} ${date.getDate()}, ${subtitle}${accessibleMoney ? `, ${accessibleMoney}` : ''}`} accessibilityState={{ disabled }}
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
        <ThemedText type="small" themeColor="textSecondary">{subtitle}</ThemedText>
      </View>

      {hasMoney && (
        <View style={styles.position}>
          <ThemedText type="small" style={[styles.positionLabel, { color: moneyColor }]}>
            {positionLabel}
          </ThemedText>
          <ThemedText style={[styles.amount, { color: moneyColor }]} numberOfLines={1} adjustsFontSizeToFit>
            {formatExpenseAmount(position.amountMinor, expense.currency)}
          </ThemedText>
        </View>
      )}
      {position.kind === 'none' && (
        <ThemedText type="small" themeColor="textSecondary" style={styles.position}>No amount owed</ThemedText>
      )}
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
  position: { maxWidth: '38%', flexShrink: 1, alignItems: 'flex-end' },
  positionLabel: { fontSize: 12, lineHeight: 18, textAlign: 'right' },
  amount: { maxWidth: '100%', fontSize: 16, lineHeight: 24, fontWeight: '600', textAlign: 'right' },
});
