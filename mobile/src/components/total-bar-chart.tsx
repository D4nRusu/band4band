import { StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import { formatExpenseAmount } from '@/lib/money';

export type BarRow = {
  key: string;
  label: string;
  icon?: string;
  amountMinor: number;
};

type TotalBarChartProps = {
  title: string;
  totalAmountMinor: number;
  rows: BarRow[];
  currency: string;
  emptyText: string;
  note?: string;
};

export function TotalBarChart({ title, totalAmountMinor, rows, currency, emptyText, note }: TotalBarChartProps) {
  const theme = useTheme();
  const largest = Math.max(0, ...rows.map((row) => row.amountMinor));

  return (
    <View style={[styles.card, { backgroundColor: theme.backgroundElement }]}>
      <View style={styles.header}>
        <ThemedText type="smallBold" accessibilityRole="header" style={styles.title}>{title}</ThemedText>
        <ThemedText type="smallBold">{formatExpenseAmount(totalAmountMinor, currency)}</ThemedText>
      </View>
      {note && <ThemedText type="small" themeColor="textSecondary">{note}</ThemedText>}

      {rows.length === 0 && <ThemedText type="small" themeColor="textSecondary">{emptyText}</ThemedText>}

      {rows.map((row) => {
        const amount = formatExpenseAmount(row.amountMinor, currency);
        const percent = totalAmountMinor > 0 ? Math.round((row.amountMinor / totalAmountMinor) * 100) : 0;
        const width = largest > 0 ? `${(row.amountMinor / largest) * 100}%` as const : '0%';

        return (
          <View key={row.key} style={styles.row} accessible accessibilityLabel={`${row.label}, ${amount}, ${percent}% of total`}>
            <View style={styles.rowText}>
              <ThemedText numberOfLines={1} style={styles.label}>
                {row.icon ? `${row.icon} ` : ''}{row.label}
              </ThemedText>
              <ThemedText type="smallBold">{amount}</ThemedText>
              <ThemedText type="small" themeColor="textSecondary" style={styles.percent}>{percent}%</ThemedText>
            </View>
            <View style={styles.track}>
              <View style={[styles.bar, { width, backgroundColor: theme.chartBar }]} />
            </View>
          </View>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  card: { borderRadius: 12, padding: 14, gap: 12 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  title: { flex: 1 },
  row: { gap: 6 },
  rowText: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  label: { flex: 1, minWidth: 0 },
  percent: { width: 40, textAlign: 'right' },
  track: { height: 8 },
  bar: { height: 8, minWidth: 2, borderTopRightRadius: 4, borderBottomRightRadius: 4 },
});
