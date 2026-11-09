import { useState } from 'react';
import {
  ActivityIndicator, Button, KeyboardAvoidingView, Modal, Platform,
  ScrollView, StyleSheet, TextInput, View,
} from 'react-native';

import { ExpenseChoices } from '@/components/expense-choices';
import { ThemedText } from '@/components/themed-text';
import { useExpenseOptions } from '@/hooks/use-expense-options';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import { categoryIcons } from '@/lib/categories-api';
import type { ExpenseInput } from '@/lib/expenses-api';
import type { Group } from '@/lib/groups-api';
import { parseEuroAmount } from '@/lib/money';

type CreateExpenseModalProps = {
  user: AuthenticatedUser & { uid: string };
  group: Group;
  saving: boolean;
  error: string | null;
  onSave: (input: ExpenseInput) => Promise<boolean>;
  onClose: () => void;
};

export function CreateExpenseModal({ user, group, saving, error, onSave, onClose }: CreateExpenseModalProps) {
  const theme = useTheme();
  const options = useExpenseOptions(user, group.id);
  const [name, setName] = useState('');
  const [amount, setAmount] = useState('');
  const [categoryID, setCategoryID] = useState('default_other');
  const [paidBy, setPaidBy] = useState(user.uid);
  const [validationError, setValidationError] = useState<string | null>(null);
  const inputStyle = [styles.input, { color: theme.text, borderColor: theme.textSecondary }];
  const busy = saving || options.loading || options.error !== null;

  function close() {
    if (!saving) onClose();
  }

  async function submit() {
    const trimmedName = name.trim();
    if (Array.from(trimmedName).length < 1 || Array.from(trimmedName).length > 100) {
      setValidationError('Enter an expense name of 1–100 characters.');
      return;
    }

    let amountMinor: number;
    try {
      amountMinor = parseEuroAmount(amount);
    } catch (cause) {
      setValidationError(cause instanceof Error ? cause.message : 'Enter a valid amount.');
      return;
    }

    if (!options.members.some((member) => member.uid === paidBy) || !options.categories.some((category) => category.id === categoryID)) {
      setValidationError('Choose an available category and current group member.');
      return;
    }

    setValidationError(null);
    if (await onSave({ name: trimmedName, amountMinor, categoryID, paidBy })) onClose();
  }

  return (
    <Modal transparent animationType="fade" onRequestClose={close}>
      <KeyboardAvoidingView style={styles.overlay} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <ThemedText type="subtitle" accessibilityRole="header">Add expense</ThemedText>
            <ThemedText>Expense name</ThemedText>
            <TextInput accessibilityLabel="Expense name" value={name} onChangeText={setName} style={inputStyle} editable={!saving} />
            <ThemedText>Amount ({group.currency})</ThemedText>
            <TextInput accessibilityLabel={`Amount in ${group.currency}`} value={amount} onChangeText={setAmount}
              style={inputStyle} editable={!saving} keyboardType="decimal-pad" placeholder="12.50" placeholderTextColor={theme.textSecondary} />

            {options.loading && <ActivityIndicator accessibilityLabel="Loading categories and members" />}
            {options.error && (
              <View style={styles.feedback}>
                <ThemedText accessibilityRole="alert">{options.error}</ThemedText>
                <Button title="Retry loading options" onPress={options.retry} disabled={options.loading || saving} />
              </View>
            )}

            {!options.loading && !options.error && (
              <>
                <ExpenseChoices label="Category" selected={categoryID} disabled={saving} onSelect={setCategoryID}
                  choices={options.categories.map((category) => ({ id: category.id, label: `${categoryIcons[category.icon].image} ${category.name}` }))} />
                <ExpenseChoices label="Paid by" selected={paidBy} disabled={saving} onSelect={setPaidBy}
                  choices={options.members.map((member) => ({ id: member.uid, label: member.name || member.username || member.uid }))} />
              </>
            )}

            {(validationError || error) && <ThemedText accessibilityRole="alert">{validationError || error}</ThemedText>}
            {saving && <ActivityIndicator accessibilityLabel="Saving expense" />}
            <Button title="Save expense" onPress={submit} disabled={busy} />
            <Button title="Cancel" onPress={close} disabled={saving} />
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: { flex: 1, justifyContent: 'center', padding: 24, backgroundColor: 'rgba(0, 0, 0, 0.5)' },
  dialog: { width: '100%', maxWidth: 600, maxHeight: '90%', alignSelf: 'center', borderRadius: 16 },
  content: { padding: 24, gap: 16 },
  feedback: { gap: 8 },
  input: { borderWidth: 1, borderRadius: 8, padding: 12, fontSize: 16 },
});
