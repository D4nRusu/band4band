import { ActivityIndicator, Button, Pressable, StyleSheet, TextInput, View } from 'react-native';

import { ErrorMessage } from '@/components/error-message';
import { ExpenseChoices } from '@/components/expense-choices';
import { ExpensePayerSelect } from '@/components/expense-payer-select';
import { ReceiptItemEditor } from '@/components/receipt-item-editor';
import { ThemedText } from '@/components/themed-text';
import { useReceiptAssignment } from '@/hooks/use-receipt-assignment';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import { categoryIcons } from '@/lib/categories-api';
import type { Group } from '@/lib/groups-api';
import { formatExpenseAmount } from '@/lib/money';
import type { ReceiptExpenseInput } from '@/lib/receipt-assignment';
import type { ReceiptDraft } from '@/lib/receipts-api';

type ReceiptAssignmentFormProps = {
  user: AuthenticatedUser & { uid: string };
  group: Group;
  draft: ReceiptDraft;
  saving: boolean;
  error: string | null;
  onSave: (input: ReceiptExpenseInput) => Promise<boolean>;
  onSaved: () => void;
};

export function ReceiptAssignmentForm({ user, group, draft, saving, error, onSave, onSaved }: ReceiptAssignmentFormProps) {
  const theme = useTheme();
  const review = useReceiptAssignment(user, group, draft);
  const { options, details } = review;
  const inputStyle = [styles.input, { color: theme.text, borderColor: theme.textSecondary }];
  const busy = saving || options.loading || options.error !== null;
  const displayedError = review.validationError || error;

  async function submit() {
    if (busy) return;

    let input: ReceiptExpenseInput;
    try {
      input = review.buildExpense();
    } catch (cause) {
      review.setValidationError(cause instanceof Error ? cause.message : 'Check the item prices and assignments.');
      return;
    }

    review.setValidationError(null);
    if (await onSave(input)) onSaved();
  }

  return (
    <View style={styles.form}>
      <ThemedText>Expense name</ThemedText>
      <TextInput value={details.name} onChangeText={(name) => review.changeDetails({ name })}
        accessibilityLabel="Receipt expense name" editable={!saving} style={inputStyle} />
      <ThemedText>Receipt total ({draft.currency})</ThemedText>
      <TextInput value={details.total} onChangeText={(total) => review.changeDetails({ total })}
        accessibilityLabel="Reviewed receipt total" keyboardType="decimal-pad" editable={!saving} style={inputStyle} />
      {review.total !== null && (
        <ThemedText type="small" themeColor="textSecondary" accessibilityLiveRegion="polite">
          Items total {review.total < 0n ? '−' : ''}{formatExpenseAmount(review.total < 0n ? -review.total : review.total, draft.currency)}
        </ThemedText>
      )}
      {review.totalsMatch === false && (
        <ErrorMessage message="The items do not add up to the receipt total. Check and correct the prices or total before saving." />
      )}

      {options.loading && <ActivityIndicator accessibilityLabel="Loading receipt expense options" />}
      {options.error && (
        <View style={styles.form}>
          <ErrorMessage message={options.error} />
          <Button title="Retry loading options" disabled={saving || options.loading} onPress={options.retry} />
        </View>
      )}

      {!options.loading && !options.error && (
        <>
          <ExpenseChoices label="Category" selected={details.categoryID} disabled={saving}
            onSelect={(categoryID) => review.changeDetails({ categoryID })}
            choices={options.categories.map((category) => ({ id: category.id, label: `${categoryIcons[category.icon].image} ${category.name}` }))} />
          <ExpensePayerSelect members={options.members} currentUID={user.uid} selectedUID={details.paidBy}
            disabled={saving} onSelect={(paidBy) => review.changeDetails({ paidBy })} />
          {review.items.map((item) => (
            <ReceiptItemEditor key={item.id} item={item} members={options.members} currentUID={user.uid}
              currency={draft.currency} disabled={saving} onChange={(patch) => review.changeItem(item.id, patch)} />
          ))}
        </>
      )}

      {draft.currency !== group.currency && (
        <Pressable accessibilityRole="checkbox" accessibilityState={{ checked: details.confirmCurrency, disabled: saving }}
          disabled={saving} onPress={() => review.changeDetails({ confirmCurrency: !details.confirmCurrency })}
          style={[styles.confirmation, { borderColor: theme.textSecondary, backgroundColor: details.confirmCurrency ? theme.backgroundSelected : theme.background }]}>
          <ThemedText type="small">
            {details.confirmCurrency ? '✓ ' : '○ '}Use these numeric amounts in {group.currency}. The receipt uses {draft.currency}; no exchange rate will be applied.
          </ThemedText>
        </Pressable>
      )}
      <ThemedText type="small" themeColor="textSecondary">
        Confirm to save an expense in {group.currency} with the assigned shares. Item prices are full line totals.
      </ThemedText>
      {displayedError && <ErrorMessage message={displayedError} />}
      {saving && <ActivityIndicator accessibilityLabel="Saving receipt expense" />}
      <Button title="Confirm and create expense" disabled={busy} onPress={submit} />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: 12 },
  input: { borderWidth: 1, borderRadius: 8, padding: 12, fontSize: 16 },
  confirmation: { padding: 12, borderWidth: 1, borderRadius: 8 },
});
