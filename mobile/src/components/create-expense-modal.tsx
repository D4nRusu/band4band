import { SymbolView } from 'expo-symbols';
import { useState } from 'react';
import {
  ActivityIndicator, Button, KeyboardAvoidingView, Modal, Platform,
  Pressable, ScrollView, StyleSheet, TextInput, View,
} from 'react-native';

import { ErrorMessage } from '@/components/error-message';
import { ExpensePayerSelect } from '@/components/expense-payer-select';
import { ExpenseParticipants } from '@/components/expense-participants';
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
  const [excludedUIDs, setExcludedUIDs] = useState<string[]>([]);
  const participantUIDs = options.members.filter((member) => !excludedUIDs.includes(member.uid)).map((member) => member.uid);
  const [validationError, setValidationError] = useState<string | null>(null);
  const inputStyle = [styles.input, { color: theme.text, borderColor: theme.textSecondary }];
  const busy = saving || options.loading || options.error !== null;
  const displayedError = validationError || error;

  function changeName(value: string) {
    setName(value);
    setValidationError(null);
  }

  function changeAmount(value: string) {
    setAmount(value);
    setValidationError(null);
  }

  function chooseCategory(id: string) {
    setCategoryID(id);
    setValidationError(null);
  }

  function choosePayer(uid: string) {
    setPaidBy(uid);
    setValidationError(null);
  }

  function toggleParticipant(uid: string) {
    setExcludedUIDs((previous) => previous.includes(uid)
      ? previous.filter((excluded) => excluded !== uid)
      : [...previous, uid]);
    setValidationError(null);
  }

  function close() {
    if (!saving) onClose();
  }

  async function submit() {
    const trimmedName = name.trim();
    if (Array.from(trimmedName).length < 1 || Array.from(trimmedName).length > 100) {
      setValidationError(trimmedName.length === 0
        ? 'Give this expense a name, such as Dinner or Groceries.'
        : 'The expense name is too long. Keep it within 100 characters.');
      return;
    }

    let amountMinor: number;
    try {
      amountMinor = parseEuroAmount(amount);
    } catch (cause) {
      setValidationError(cause instanceof Error ? cause.message : 'Enter an amount greater than €0, using at most two decimal places.');
      return;
    }

    if (!options.categories.some((category) => category.id === categoryID)) {
      setValidationError('Choose a category from the list. If it was removed, reopen this form to load the current categories.');
      return;
    }

    if (!options.members.some((member) => member.uid === paidBy)) {
      setValidationError('Choose who paid from the current group members. Reopen this form if someone has left the group.');
      return;
    }

    if (participantUIDs.length === 0) {
      setValidationError('Nobody is included in the split. Select at least one person under “Split equally between”.');
      return;
    }

    setValidationError(null);
    if (await onSave({ name: trimmedName, amountMinor, categoryID, paidBy, participantUIDs })) onClose();
  }

  return (
    <Modal transparent animationType="fade" onRequestClose={close}>
      <KeyboardAvoidingView style={styles.overlay} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <View style={styles.header}>
            <Pressable onPress={close} disabled={saving} accessibilityRole="button"
              accessibilityLabel="Cancel adding expense" accessibilityState={{ disabled: saving }}
              style={({ pressed }) => [styles.back, pressed && styles.pressed, saving && styles.disabled]}>
              <SymbolView name={{ ios: 'arrow.left', android: 'arrow_back', web: 'arrow_back' }} size={24} tintColor={theme.text} />
            </Pressable>
            <ThemedText type="subtitle" style={styles.title} accessibilityRole="header">Add expense</ThemedText>
          </View>

          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <ThemedText>Expense name</ThemedText>
            <TextInput accessibilityLabel="Expense name" value={name} onChangeText={changeName} style={inputStyle} editable={!saving} />
            <ThemedText>Amount ({group.currency})</ThemedText>
            <TextInput accessibilityLabel={`Amount in ${group.currency}`} value={amount} onChangeText={changeAmount}
              style={inputStyle} editable={!saving} keyboardType="decimal-pad" placeholder="12.50" placeholderTextColor={theme.textSecondary} />

            {options.loading && <ActivityIndicator accessibilityLabel="Loading categories and members" />}
            {options.error && (
              <View style={styles.feedback}>
                <ErrorMessage message={options.error} />
                <Button title="Retry loading options" onPress={options.retry} disabled={options.loading || saving} />
              </View>
            )}

            {!options.loading && !options.error && (
              <>
                <ExpenseChoices label="Category" selected={categoryID} disabled={saving} onSelect={chooseCategory}
                  choices={options.categories.map((category) => ({ id: category.id, label: `${categoryIcons[category.icon].image} ${category.name}` }))} />
                <ExpensePayerSelect members={options.members} selectedUID={paidBy} currentUID={user.uid}
                  disabled={saving} onSelect={choosePayer} />
                <ExpenseParticipants members={options.members} selectedUIDs={participantUIDs}
                  disabled={saving} onToggle={toggleParticipant} />
              </>
            )}

            {displayedError && <ErrorMessage message={displayedError} />}
            {saving && <ActivityIndicator accessibilityLabel="Saving expense" />}
            <Button title="Save expense" onPress={submit} disabled={busy} />
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: { flex: 1, justifyContent: 'center', padding: 24, backgroundColor: 'rgba(0, 0, 0, 0.5)' },
  dialog: { width: '100%', maxWidth: 600, maxHeight: '90%', alignSelf: 'center', borderRadius: 16 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 8, padding: 12 },
  back: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center', borderRadius: 24 },
  title: { flex: 1, fontSize: 24, lineHeight: 32 },
  pressed: { opacity: 0.7 },
  disabled: { opacity: 0.5 },
  content: { padding: 24, paddingTop: 8, gap: 16 },
  feedback: { gap: 8 },
  input: { borderWidth: 1, borderRadius: 8, padding: 12, fontSize: 16 },
});
