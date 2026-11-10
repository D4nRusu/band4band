import { SymbolView } from 'expo-symbols';
import { useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Button,
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  TextInput,
  View,
} from 'react-native';

import { ErrorMessage } from '@/components/error-message';
import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import { parseEuroAmount } from '@/lib/money';
import type { GroupMember } from '@/lib/groups-api';
import type { SettleUpInput } from '@/lib/settlements-api';

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

type SettleUpModalProps = {
  currentUID: string;
  currency: string;
  members: GroupMember[];
  membersLoading: boolean;
  membersError: string | null;
  debts: DebtRow[];
  receivables: ReceivableRow[];
  saving: boolean;
  error: string | null;
  onRetryMembers: () => void;
  onSettle: (input: SettleUpInput) => Promise<boolean>;
  onClose: () => void;
};

export function SettleUpModal({
  currentUID,
  currency,
  members,
  membersLoading,
  membersError,
  debts,
  receivables,
  saving,
  error,
  onRetryMembers,
  onSettle,
  onClose,
}: SettleUpModalProps) {
  const theme = useTheme();
  const [fromUID, setFromUID] = useState<string | null>(currentUID);
  const [toUID, setToUID] = useState<string | null>(null);
  const [amount, setAmount] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);

  const debtByPair = useMemo(() => {
    const values = new Map<string, number>();
    debts.forEach((debt) => values.set(pairKey(currentUID, debt.toUID), debt.amountMinor));
    receivables.forEach((debt) => values.set(pairKey(debt.fromUID, currentUID), debt.amountMinor));
    return values;
  }, [currentUID, debts, receivables]);

  const busy = saving || membersLoading || membersError !== null;
  const confirmDisabled = busy || fromUID === null || toUID === null || isZeroAmount(amount);
  const displayedError = validationError || error;

  function close() {
    if (!saving) onClose();
  }

  function selectPair(nextFrom: string | null, nextTo: string | null) {
    setFromUID(nextFrom);
    setToUID(nextTo);
    setValidationError(null);

    if (nextFrom && nextTo) {
      setAmount(minorToAmountInput(debtByPair.get(pairKey(nextFrom, nextTo)) ?? 0));
    }
  }

  // One side of a settlement must be the current user, so picking someone else forces the other side to "You".
  function choosePayer(uid: string) {
    if (uid !== currentUID) {
      selectPair(uid, currentUID);
      return;
    }

    selectPair(uid, toUID === currentUID ? null : toUID);
  }

  function chooseRecipient(uid: string) {
    if (uid !== currentUID) {
      selectPair(currentUID, uid);
      return;
    }

    selectPair(fromUID === currentUID ? null : fromUID, uid);
  }

  function changeAmount(value: string) {
    setAmount(value);
    setValidationError(null);
  }

  async function submit() {
    if (!fromUID || !toUID) {
      setValidationError('Choose who paid and who received the payment.');
      return;
    }

    let amountMinor: number;
    try {
      amountMinor = parseEuroAmount(amount);
    } catch (cause) {
      setValidationError(cause instanceof Error ? cause.message : 'Enter a valid amount greater than 0.');
      return;
    }

    setValidationError(null);
    if (await onSettle({ fromUID, toUID, amountMinor })) onClose();
  }

  return (
    <Modal transparent animationType="fade" onRequestClose={close}>
      <KeyboardAvoidingView style={styles.overlay} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <View style={styles.header}>
            <Pressable
              onPress={close}
              disabled={saving}
              accessibilityRole="button"
              accessibilityLabel="Cancel settle up"
              accessibilityState={{ disabled: saving }}
              style={({ pressed }) => [styles.back, pressed && styles.pressed, saving && styles.disabled]}
            >
              <SymbolView name={{ ios: 'arrow.left', android: 'arrow_back', web: 'arrow_back' }} size={24} tintColor={theme.text} />
            </Pressable>
            <ThemedText type="subtitle" style={styles.title} accessibilityRole="header">Settle up</ThemedText>
          </View>

          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            {membersLoading && <ActivityIndicator accessibilityLabel="Loading group members" />}
            {membersError && (
              <View style={styles.feedback}>
                <ErrorMessage message={membersError} />
                <Button title="Retry members" onPress={onRetryMembers} disabled={membersLoading || saving} />
              </View>
            )}

            {!membersLoading && !membersError && (
              <>
                <MemberPicker
                  label="Paid by"
                  members={members}
                  currentUID={currentUID}
                  selectedUID={fromUID}
                  disabled={saving}
                  onSelect={choosePayer}
                />
                <MemberPicker
                  label="Paid to"
                  members={members}
                  currentUID={currentUID}
                  selectedUID={toUID}
                  disabled={saving}
                  onSelect={chooseRecipient}
                />
              </>
            )}

            <ThemedText>Amount ({currency})</ThemedText>
            <TextInput
              accessibilityLabel={`Amount in ${currency}`}
              value={amount}
              onChangeText={changeAmount}
              style={[styles.input, { color: theme.text, borderColor: theme.textSecondary }]}
              editable={!saving}
              keyboardType="decimal-pad"
              placeholder="12.50"
              placeholderTextColor={theme.textSecondary}
            />

            <View style={styles.quickSection}>
              <ThemedText type="smallBold">Your debts</ThemedText>
              {debts.length === 0 && receivables.length === 0 && (
                <ThemedText type="small" themeColor="textSecondary">You are settled up with everyone in this group.</ThemedText>
              )}

              {debts.map((debt) => (
                <QuickDebt
                  key={`owe-${debt.toUID}`}
                  label={`You owe ${debt.toName}`}
                  amount={formatMoney(debt.amountMinor, currency)}
                  disabled={saving}
                  onPress={() => selectPair(currentUID, debt.toUID)}
                />
              ))}

              {receivables.map((debt) => (
                <QuickDebt
                  key={`owed-${debt.fromUID}`}
                  label={`${debt.fromName} owes you`}
                  amount={formatMoney(debt.amountMinor, currency)}
                  disabled={saving}
                  onPress={() => selectPair(debt.fromUID, currentUID)}
                />
              ))}
            </View>

            {displayedError && <ErrorMessage message={displayedError} />}
            {saving && <ActivityIndicator accessibilityLabel="Saving settlement" />}
            <Button title="Confirm transfer" onPress={submit} disabled={confirmDisabled} />
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

type MemberPickerProps = {
  label: string;
  members: GroupMember[];
  currentUID: string;
  selectedUID: string | null;
  disabled: boolean;
  onSelect: (uid: string) => void;
};

function MemberPicker({ label, members, currentUID, selectedUID, disabled, onSelect }: MemberPickerProps) {
  const theme = useTheme();

  return (
    <View style={styles.picker}>
      <ThemedText>{label}</ThemedText>
      <View style={styles.chips} accessibilityRole="radiogroup" accessibilityLabel={label}>
        {members.map((member) => {
          const selected = member.uid === selectedUID;
          const name = member.uid === currentUID ? 'You' : member.name;

          return (
            <Pressable
              key={member.uid}
              onPress={() => onSelect(member.uid)}
              disabled={disabled}
              accessibilityRole="radio"
              accessibilityLabel={`${label} ${name}`}
              accessibilityState={{ selected, disabled }}
              style={({ pressed }) => [
                styles.chip,
                { borderColor: selected ? theme.text : theme.textSecondary },
                selected && { backgroundColor: theme.backgroundElement },
                pressed && styles.pressed,
              ]}
            >
              <ThemedText type={selected ? 'smallBold' : 'small'}>{name}</ThemedText>
            </Pressable>
          );
        })}
      </View>
    </View>
  );
}

type QuickDebtProps = {
  label: string;
  amount: string;
  disabled: boolean;
  onPress: () => void;
};

function QuickDebt({ label, amount, disabled, onPress }: QuickDebtProps) {
  const theme = useTheme();

  return (
    <Pressable
      onPress={onPress}
      disabled={disabled}
      accessibilityRole="button"
      style={({ pressed }) => [styles.quickDebt, { backgroundColor: theme.backgroundElement }, pressed && styles.pressed]}
    >
      <ThemedText>{label}</ThemedText>
      <ThemedText type="smallBold">{amount}</ThemedText>
    </Pressable>
  );
}

function pairKey(fromUID: string, toUID: string): string {
  return `${fromUID}->${toUID}`;
}

function isZeroAmount(value: string): boolean {
  return /^[\s0.,]*$/.test(value);
}

function minorToAmountInput(amountMinor: number): string {
  const whole = Math.floor(amountMinor / 100);
  const fraction = Math.abs(amountMinor % 100).toString().padStart(2, '0');
  return `${whole}.${fraction}`;
}

function formatMoney(amountMinor: number, currency: string): string {
  const whole = Math.floor(amountMinor / 100);
  const fraction = Math.abs(amountMinor % 100).toString().padStart(2, '0');
  return currency === 'EUR' ? `EUR ${whole}.${fraction}` : `${whole}.${fraction} ${currency}`;
}

const styles = StyleSheet.create({
  overlay: { flex: 1, justifyContent: 'center', padding: 24, backgroundColor: 'rgba(0, 0, 0, 0.5)' },
  dialog: { width: '100%', maxWidth: 600, maxHeight: '90%', alignSelf: 'center', borderRadius: 16 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 8, padding: 12 },
  back: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center', borderRadius: 24 },
  title: { flex: 1, fontSize: 24, lineHeight: 32 },
  content: { padding: 24, paddingTop: 8, gap: 16 },
  input: { borderWidth: 1, borderRadius: 8, padding: 12, fontSize: 16 },
  feedback: { gap: 8 },
  picker: { gap: 8 },
  chips: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  chip: { minHeight: 40, paddingHorizontal: 14, justifyContent: 'center', borderWidth: 1, borderRadius: 20 },
  quickSection: { gap: 8 },
  quickDebt: {
    paddingHorizontal: 12,
    paddingVertical: 10,
    borderRadius: 10,
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  pressed: { opacity: 0.7 },
  disabled: { opacity: 0.5 },
});
