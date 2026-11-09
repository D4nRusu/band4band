import { SymbolView } from 'expo-symbols';
import { ActivityIndicator, Alert, Modal, Pressable, ScrollView, StyleSheet, View } from 'react-native';

import { ExpenseContributions } from '@/components/expense-contributions';
import { ErrorMessage } from '@/components/error-message';
import { ThemedText } from '@/components/themed-text';
import { useDeleteExpense } from '@/hooks/use-delete-expense';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import { categoryIcons } from '@/lib/categories-api';
import type { Expense } from '@/lib/expenses-api';
import { formatExpenseAmount } from '@/lib/money';

type ExpenseDetailsModalProps = {
  user: AuthenticatedUser;
  expense: Expense;
  onClose: () => void;
  onDeleted: () => void;
};

export function ExpenseDetailsModal({ user, expense, onClose, onDeleted }: ExpenseDetailsModalProps) {
  const theme = useTheme();
  const { deleting, error, remove } = useDeleteExpense(user, expense);
  const addedOn = new Date(expense.createdAt).toLocaleDateString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
  });

  function close() {
    if (!deleting) onClose();
  }

  async function deleteConfirmed() {
    if (await remove()) onDeleted();
  }

  function confirmDelete() {
    Alert.alert('Delete expense?', `Delete “${expense.name}” from this group?`, [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Delete', style: 'destructive', onPress: () => void deleteConfirmed() },
    ]);
  }

  return (
    <Modal transparent animationType="fade" onRequestClose={close}>
      <View style={styles.overlay}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <View style={styles.header}>
            <Pressable onPress={close} disabled={deleting} accessibilityRole="button"
              accessibilityLabel="Back to expenses" style={styles.iconButton}>
              <SymbolView name={{ ios: 'arrow.left', android: 'arrow_back', web: 'arrow_back' }} size={24} tintColor={theme.text} />
            </Pressable>
            <ThemedText style={styles.heading} type="smallBold" accessibilityRole="header">Expense details</ThemedText>
            <Pressable onPress={confirmDelete} disabled={deleting} accessibilityRole="button"
              accessibilityLabel="Delete expense" accessibilityState={{ disabled: deleting }} style={styles.iconButton}>
              <SymbolView name={{ ios: 'trash', android: 'delete', web: 'delete' }} size={22} tintColor="#d32f2f" />
            </Pressable>
          </View>

          <ScrollView contentContainerStyle={styles.content}>
            <View style={styles.summary}>
              <View style={styles.titleRow}>
                <ThemedText type="subtitle" style={styles.title}>{expense.name}</ThemedText>
                <ThemedText style={styles.amount} numberOfLines={1} adjustsFontSizeToFit>
                  {formatExpenseAmount(expense.amountMinor, expense.currency)}
                </ThemedText>
              </View>
              <ThemedText type="small" themeColor="textSecondary">
                {categoryIcons[expense.categoryIcon].image} {expense.categoryName}
              </ThemedText>
              <ThemedText type="small" themeColor="textSecondary">Paid by {expense.payerName} on {addedOn}</ThemedText>
            </View>

            {deleting && <ActivityIndicator accessibilityLabel="Deleting expense" />}
            {error && <ErrorMessage message={error} />}

            <ExpenseContributions expense={expense} />
          </ScrollView>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: { flex: 1, justifyContent: 'center', padding: 24, backgroundColor: 'rgba(0, 0, 0, 0.5)' },
  dialog: { width: '100%', maxWidth: 600, maxHeight: '90%', alignSelf: 'center', borderRadius: 16 },
  header: { flexDirection: 'row', alignItems: 'center', padding: 12, gap: 8 },
  heading: { flex: 1 },
  iconButton: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center' },
  content: { padding: 24, paddingTop: 8, gap: 16 },
  summary: { gap: 6 },
  titleRow: { flexDirection: 'row', alignItems: 'baseline', gap: 12 },
  title: { flex: 1, minWidth: 0 },
  amount: { maxWidth: '45%', fontSize: 24, lineHeight: 32, fontWeight: '600', textAlign: 'right' },
});
