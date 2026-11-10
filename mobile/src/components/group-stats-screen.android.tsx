import { router, useLocalSearchParams } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { ActivityIndicator, Button, Pressable, RefreshControl, ScrollView, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { TotalBarChart, type BarRow } from '@/components/total-bar-chart';
import { ThemedText } from '@/components/themed-text';
import { useAuthSession } from '@/hooks/use-auth-session.android';
import { useGroupDetails } from '@/hooks/use-group-details';
import { useGroupStats } from '@/hooks/use-group-stats';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import { categoryIcons } from '@/lib/categories-api';
import type { CategoryTotal } from '@/lib/stats-api';

export default function GroupStatsScreen() {
  const theme = useTheme();
  const { groupID } = useLocalSearchParams<{ groupID: string }>();
  const { user, initializing } = useAuthSession();
  const validID = typeof groupID === 'string' && groupID.length > 0;

  return (
    <SafeAreaView style={[styles.screen, { backgroundColor: theme.background }]}>
      {initializing && <ActivityIndicator accessibilityLabel="Restoring session" />}
      {!initializing && !user && <ThemedText>Sign in from Home to view group statistics.</ThemedText>}
      {user && !validID && <ThemedText accessibilityRole="alert">This group link is invalid.</ThemedText>}
      {user && validID && <GroupStats key={`${user.uid}:${groupID}`} user={user} groupID={groupID} />}
    </SafeAreaView>
  );
}

function GroupStats({ user, groupID }: { user: AuthenticatedUser & { uid: string }; groupID: string }) {
  const theme = useTheme();
  const { group } = useGroupDetails(user, groupID);
  const { stats, loading, error, reload } = useGroupStats(user, groupID);
  const currency = group?.currency ?? 'EUR';

  function back() {
    router.dismissTo({ pathname: '/groups/[groupID]', params: { groupID } });
  }

  const memberRows: BarRow[] = (stats?.byMember ?? []).map((member) => ({
    key: member.uid,
    label: member.uid === user.uid ? 'You' : member.name,
    amountMinor: member.amountMinor,
  }));
  const memberTotal = memberRows.reduce((sum, row) => sum + row.amountMinor, 0);

  return (
    <>
      <View style={styles.header}>
        <Pressable onPress={back} accessibilityRole="button" accessibilityLabel="Back to group" style={styles.back}>
          <SymbolView name={{ ios: 'arrow.left', android: 'arrow_back', web: 'arrow_back' }} size={24} tintColor={theme.text} />
        </Pressable>
        <ThemedText type="subtitle" style={styles.title} accessibilityRole="header" numberOfLines={1}>
          {group ? `${group.name} stats` : 'Stats'}
        </ThemedText>
      </View>

      <ScrollView
        contentContainerStyle={styles.content}
        refreshControl={<RefreshControl refreshing={loading} onRefresh={reload} />}
      >
        {error && (
          <View style={styles.feedback}>
            <ThemedText accessibilityRole="alert">{error}</ThemedText>
            <Button title="Retry" onPress={reload} disabled={loading} />
          </View>
        )}

        {stats && (
          <>
            <TotalBarChart
              title="Group spending by category"
              totalAmountMinor={stats.totalAmountMinor}
              rows={categoryRows(stats.byCategory)}
              currency={currency}
              emptyText="No expenses yet."
            />
            <TotalBarChart
              title="Group spending by person"
              note="Each person's share of the expenses they took part in."
              totalAmountMinor={memberTotal}
              rows={memberRows}
              currency={currency}
              emptyText="No shared expenses yet."
            />
            <TotalBarChart
              title="Your spending by category"
              note="Your share of each expense."
              totalAmountMinor={stats.personalTotalAmountMinor}
              rows={categoryRows(stats.personalByCategory)}
              currency={currency}
              emptyText="You are not part of any expense yet."
            />
          </>
        )}
      </ScrollView>
    </>
  );
}

function categoryRows(totals: CategoryTotal[]): BarRow[] {
  return totals.map((total) => ({
    key: total.categoryID,
    label: total.categoryName,
    icon: categoryIcons[total.categoryIcon].image,
    amountMinor: total.amountMinor,
  }));
}

const styles = StyleSheet.create({
  screen: { flex: 1, paddingHorizontal: 24 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingBottom: 8 },
  back: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center' },
  title: { flex: 1 },
  content: { gap: 16, paddingBottom: 24 },
  feedback: { gap: 12 },
});
