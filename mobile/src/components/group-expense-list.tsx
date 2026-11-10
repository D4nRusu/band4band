import { ActivityIndicator, Button, Pressable, StyleSheet, View } from 'react-native';

import { ExpenseCard } from '@/components/expense-card';
import { ErrorMessage } from '@/components/error-message';
import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { Expense } from '@/lib/expenses-api';
import { formatExpenseAmount } from '@/lib/money';

type DebtRow = {
  toUID: string;
  toName: string;
  amountMinor: number;
};

type ReceivableRow = {
  fromUID: string;
  fromName: string;
  amountMinor: number;
};

type GroupExpenseListProps = {
  expenses: Expense[];
  debts: DebtRow[];
  receivables: ReceivableRow[];
  debtCurrency: string;
  debtsLoading: boolean;
  debtsError: string | null;
  loading: boolean;
  error: string | null;
  saving: boolean;
  settleDisabled: boolean;
  onAdd: () => void;
  onSettleUp: () => void;
  onOpen: (expense: Expense) => void;
  onRetry: () => void;
  onRetryDebts: () => void;
};

export function GroupExpenseList({
  expenses,
  debts,
  receivables,
  debtCurrency,
  debtsLoading,
  debtsError,
  loading,
  error,
  saving,
  settleDisabled,
  onAdd,
  onSettleUp,
  onOpen,
  onRetry,
  onRetryDebts,
}: GroupExpenseListProps) {
  const theme = useTheme();
  const addDisabled = loading || saving || error !== null;

  return (
    <View style={styles.section}>
      <View style={[styles.debtCard, { backgroundColor: theme.backgroundElement }]}>
        <ThemedText type="smallBold">Your debts in this group</ThemedText>
        {debtsLoading && <ActivityIndicator accessibilityLabel="Loading debt summary" />}
        {debtsError && (
          <View style={styles.feedback}>
            <ErrorMessage message={debtsError} />
            <Button title="Retry debt summary" onPress={onRetryDebts} disabled={debtsLoading || saving} />
          </View>
        )}
        {!debtsLoading && !debtsError && (
          <>
            <ThemedText type="smallBold" themeColor="textSecondary">You owe</ThemedText>
            {debts.length === 0 && (
              <ThemedText type="small" themeColor="textSecondary">You do not owe anything.</ThemedText>
            )}
            {debts.map((debt) => (
              <View key={debt.toUID} style={styles.debtRow}>
                <ThemedText>{debt.toName}</ThemedText>
                <ThemedText type="smallBold">{formatExpenseAmount(debt.amountMinor, debtCurrency)}</ThemedText>
              </View>
            ))}

            <ThemedText type="smallBold" themeColor="textSecondary">You are owed</ThemedText>
            {receivables.length === 0 && (
              <ThemedText type="small" themeColor="textSecondary">Nobody currently owes you.</ThemedText>
            )}
            {receivables.map((debt) => (
              <View key={debt.fromUID} style={styles.debtRow}>
                <ThemedText>{debt.fromName}</ThemedText>
                <ThemedText type="smallBold">{formatExpenseAmount(debt.amountMinor, debtCurrency)}</ThemedText>
              </View>
            ))}

            {debts.length === 0 && receivables.length === 0 && (
              <ThemedText type="small" themeColor="textSecondary">You are settled up with everyone.</ThemedText>
            )}
          </>
        )}
      </View>

      <View style={styles.actions}>
        <Pressable onPress={onAdd} disabled={addDisabled} accessibilityRole="button"
          accessibilityLabel="Add expense" accessibilityState={{ disabled: addDisabled }}
          style={({ pressed }) => [styles.action, { backgroundColor: theme.backgroundElement },
            pressed && styles.pressed, addDisabled && styles.disabled]}>
          <ThemedText style={styles.plus} accessible={false}>+</ThemedText>
          <ThemedText type="smallBold">Add expense</ThemedText>
        </Pressable>

        <Pressable onPress={onSettleUp} disabled={settleDisabled} accessibilityRole="button"
          accessibilityLabel="Settle up" accessibilityState={{ disabled: settleDisabled }}
          style={({ pressed }) => [styles.action, { backgroundColor: theme.backgroundElement },
            pressed && styles.pressed, settleDisabled && styles.disabled]}>
          <ThemedText type="smallBold">Settle up</ThemedText>
        </Pressable>
      </View>

      {loading && <ActivityIndicator accessibilityLabel="Loading expenses" />}
      {error && (
        <View style={styles.feedback}>
          <ErrorMessage message={error} />
          <Button title="Retry expenses" onPress={onRetry} disabled={loading || saving} />
        </View>
      )}
      {!loading && !error && expenses.length === 0 && <ThemedText>No expenses yet. Add the first one above.</ThemedText>}

      {expenses.map((expense) => (
        <ExpenseCard key={expense.id} expense={expense} onOpen={onOpen} disabled={saving} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { paddingHorizontal: 24, paddingTop: 12, paddingBottom: 24, gap: 16 },
  debtCard: { borderRadius: 12, padding: 14, gap: 8 },
  debtRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', gap: 12 },
  actions: { flexDirection: 'row', alignItems: 'center', gap: 8, flexWrap: 'wrap' },
  action: { alignSelf: 'flex-start', flexDirection: 'row', alignItems: 'center', gap: 8,
    minHeight: 48, paddingHorizontal: 14, borderRadius: 24 },
  plus: { fontSize: 24, lineHeight: 28 },
  pressed: { opacity: 0.7 },
  disabled: { opacity: 0.5 },
  feedback: { gap: 8 },
});
