import { StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { Expense } from '@/lib/expenses-api';
import { formatExpenseAmount } from '@/lib/money';

export function ExpenseContributions({ expense }: { expense: Expense }) {
  const theme = useTheme();
  const contributions = expense.contributions ?? [];

  if (contributions.length === 0) {
    return <ThemedText themeColor="textSecondary">No contributions</ThemedText>;
  }

  return (
    <View style={styles.list}>
      {contributions.map((contribution) => (
        <View key={contribution.uid} style={[styles.row, { backgroundColor: theme.backgroundElement }]}>
          <View style={styles.person}>
            <ThemedText>{contribution.name}</ThemedText>
            <ThemedText type="small" themeColor="textSecondary">
              {contribution.uid === expense.paidBy ? 'Own share' : 'Owes'}
            </ThemedText>
          </View>
          <ThemedText type="smallBold">{formatExpenseAmount(contribution.amountMinor, expense.currency)}</ThemedText>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  list: { gap: 8, marginTop: 8 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 12, padding: 12, borderRadius: 12 },
  person: { flex: 1, gap: 2 },
});
