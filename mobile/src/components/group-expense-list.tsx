import { useMemo, useState } from 'react';
import { ActivityIndicator, Button, Pressable, StyleSheet, View } from 'react-native';

import { ExpenseCard } from '@/components/expense-card';
import { ErrorMessage } from '@/components/error-message';
import { SettlementCard } from '@/components/settlement-card';
import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { Expense } from '@/lib/expenses-api';
import { formatExpenseAmount } from '@/lib/money';
import type { Settlement } from '@/lib/settlements-api';

type HistoryFilter = 'all' | 'expenses' | 'settlements';

type HistoryEntry =
  | { kind: 'expense'; createdAt: string; expense: Expense }
  | { kind: 'settlement'; createdAt: string; settlement: Settlement };

const historyFilters: { value: HistoryFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'expenses', label: 'Expenses' },
  { value: 'settlements', label: 'Settlements' },
];

const emptyHistoryText: Record<HistoryFilter, string> = {
  all: 'No expenses or settlements yet. Add the first expense above.',
  expenses: 'No expenses yet. Add the first one above.',
  settlements: 'No settlements yet.',
};

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
  currentUID: string;
  expenses: Expense[];
  settlements: Settlement[];
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
  currentUID,
  expenses,
  settlements,
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
  const [filter, setFilter] = useState<HistoryFilter>('all');
  const history = useMemo(() => buildHistory(expenses, settlements, filter), [expenses, settlements, filter]);

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

      <HistoryFilterPicker value={filter} onChange={setFilter} />

      {loading && <ActivityIndicator accessibilityLabel="Loading expenses" />}
      {error && (
        <View style={styles.feedback}>
          <ErrorMessage message={error} />
          <Button title="Retry expenses" onPress={onRetry} disabled={loading || saving} />
        </View>
      )}
      {!loading && !error && history.length === 0 && <ThemedText>{emptyHistoryText[filter]}</ThemedText>}

      {history.map((entry) => (entry.kind === 'expense'
        ? <ExpenseCard key={`expense-${entry.expense.id}`} expense={entry.expense} onOpen={onOpen} disabled={saving} />
        : <SettlementCard key={`settlement-${entry.settlement.id}`} settlement={entry.settlement} currentUID={currentUID} />
      ))}
    </View>
  );
}

function HistoryFilterPicker({ value, onChange }: { value: HistoryFilter; onChange: (value: HistoryFilter) => void }) {
  const theme = useTheme();

  return (
    <View style={styles.filters} accessibilityRole="radiogroup" accessibilityLabel="Show in history">
      {historyFilters.map((option) => {
        const selected = option.value === value;

        return (
          <Pressable
            key={option.value}
            onPress={() => onChange(option.value)}
            accessibilityRole="radio"
            accessibilityState={{ selected }}
            style={({ pressed }) => [
              styles.filter,
              { borderColor: selected ? theme.text : theme.textSecondary },
              selected && { backgroundColor: theme.backgroundElement },
              pressed && styles.pressed,
            ]}
          >
            <ThemedText type={selected ? 'smallBold' : 'small'}>{option.label}</ThemedText>
          </Pressable>
        );
      })}
    </View>
  );
}

function buildHistory(expenses: Expense[], settlements: Settlement[], filter: HistoryFilter): HistoryEntry[] {
  const entries: HistoryEntry[] = [];

  if (filter !== 'settlements') {
    expenses.forEach((expense) => entries.push({ kind: 'expense', createdAt: expense.createdAt, expense }));
  }

  if (filter !== 'expenses') {
    settlements.forEach((settlement) => entries.push({ kind: 'settlement', createdAt: settlement.createdAt, settlement }));
  }

  return entries.sort((left, right) => Date.parse(right.createdAt) - Date.parse(left.createdAt));
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
  filters: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  filter: { minHeight: 40, paddingHorizontal: 14, justifyContent: 'center', borderWidth: 1, borderRadius: 20 },
});
