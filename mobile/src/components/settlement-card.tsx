import { StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import { formatExpenseAmount } from '@/lib/money';
import type { Settlement } from '@/lib/settlements-api';

type SettlementCardProps = {
  settlement: Settlement;
  currentUID: string;
};

export function SettlementCard({ settlement, currentUID }: SettlementCardProps) {
  const theme = useTheme();
  const date = new Date(settlement.createdAt);
  const month = date.toLocaleString('en-US', { month: 'short' });
  const amount = formatExpenseAmount(settlement.amountMinor, settlement.currency);

  const fromName = settlement.fromUID === currentUID ? 'You' : settlement.fromName;
  const toName = settlement.toUID === currentUID ? 'you' : settlement.toName;
  const title = `${fromName} paid ${toName}`;

  let positionLabel = 'Settlement';
  let amountColor: string = theme.text;
  if (settlement.fromUID === currentUID) {
    positionLabel = 'You paid';
    amountColor = theme.moneyReceivable;
  } else if (settlement.toUID === currentUID) {
    positionLabel = 'You received';
    amountColor = theme.moneyOwed;
  }

  return (
    <View
      accessible
      accessibilityLabel={`Settlement, ${month} ${date.getDate()}, ${title} ${amount}`}
      style={[styles.card, { backgroundColor: theme.backgroundElement }]}
    >
      <View style={styles.date}>
        <ThemedText type="small" themeColor="textSecondary">{month}.</ThemedText>
        <ThemedText type="small" themeColor="textSecondary">{date.getDate()}</ThemedText>
      </View>
      <ThemedText style={styles.icon} accessible={false}>💸</ThemedText>

      <View style={styles.details}>
        <ThemedText style={styles.title} numberOfLines={1}>{title}</ThemedText>
        <ThemedText type="small" themeColor="textSecondary">Settlement</ThemedText>
      </View>

      <View style={styles.position}>
        <ThemedText type="small" style={[styles.positionLabel, { color: amountColor }]}>{positionLabel}</ThemedText>
        <ThemedText style={[styles.amount, { color: amountColor }]} numberOfLines={1} adjustsFontSizeToFit>
          {amount}
        </ThemedText>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flexDirection: 'row', alignItems: 'center', padding: 12, gap: 8, minHeight: 72, borderRadius: 12 },
  date: { width: 36, alignItems: 'center' },
  icon: { width: 32, fontSize: 28, lineHeight: 36, textAlign: 'center' },
  details: { flex: 1, minWidth: 0, gap: 2 },
  title: { fontSize: 18, lineHeight: 24, fontWeight: '600' },
  position: { maxWidth: '38%', flexShrink: 1, alignItems: 'flex-end' },
  positionLabel: { fontSize: 12, lineHeight: 18, textAlign: 'right' },
  amount: { maxWidth: '100%', fontSize: 16, lineHeight: 24, fontWeight: '600', textAlign: 'right' },
});
