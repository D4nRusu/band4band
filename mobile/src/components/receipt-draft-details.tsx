import { StyleSheet, View } from 'react-native';

import { ErrorMessage } from '@/components/error-message';
import { ThemedText } from '@/components/themed-text';
import { formatExpenseAmount } from '@/lib/money';
import type { ReceiptDraft } from '@/lib/receipts-api';

export function ReceiptDraftDetails({ draft }: { draft: ReceiptDraft }) {
  return (
    <View style={styles.section}>
      <View style={styles.row}>
        <ThemedText type="smallBold">Receipt total</ThemedText>
        <ThemedText type="smallBold">{formatExpenseAmount(draft.totalMinor, draft.currency)}</ThemedText>
      </View>
      {!draft.totalsMatch && (
        <ErrorMessage message={`The items add up to ${signedAmount(draft.itemsTotalMinor, draft.currency)}, which differs from the receipt total. Check the photo and extracted prices.`} />
      )}
      {draft.items.map((item) => (
        <View key={item.id} style={styles.row}>
          <View style={styles.item}>
            <ThemedText>{item.name}</ThemedText>
            {typeof item.quantity === 'number' && (
              <ThemedText type="small" themeColor="textSecondary">Qty {item.quantity}</ThemedText>
            )}
          </View>
          <ThemedText>{signedAmount(item.amountMinor, draft.currency)}</ThemedText>
        </View>
      ))}
      <ThemedText type="small" themeColor="textSecondary">
        Check these items against the photo. This is a preview; no expense or receipt has been saved.
      </ThemedText>
    </View>
  );
}

function signedAmount(amount: number, currency: string): string {
  return amount < 0 ? `−${formatExpenseAmount(-amount, currency)}` : formatExpenseAmount(amount, currency);
}

const styles = StyleSheet.create({
  section: { gap: 16 },
  row: { flexDirection: 'row', alignItems: 'flex-start', justifyContent: 'space-between', gap: 16 },
  item: { flex: 1 },
});
