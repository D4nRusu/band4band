import { router, useLocalSearchParams } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { useState } from 'react';
import { ActivityIndicator, Button, FlatList, Pressable, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { GroupMemberRow } from '@/components/group-member-row';
import { InviteUserModal } from '@/components/invite-user-modal';
import { ThemedText } from '@/components/themed-text';
import { useAuthSession } from '@/hooks/use-auth-session.android';
import { useGroupBalances } from '@/hooks/use-group-balances';
import { useGroupDetails } from '@/hooks/use-group-details';
import { useGroupMembers } from '@/hooks/use-group-members';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import type { GroupMember } from '@/lib/groups-api';
import { formatExpenseAmount } from '@/lib/money';

export default function GroupPeopleScreen() {
  const theme = useTheme();
  const { groupID } = useLocalSearchParams<{ groupID: string }>();
  const { user, initializing } = useAuthSession();
  const validID = typeof groupID === 'string' && groupID.length > 0;

  return (
    <SafeAreaView style={[styles.screen, { backgroundColor: theme.background }]}>
      {initializing && <ActivityIndicator accessibilityLabel="Restoring session" />}
      {!initializing && !user && <ThemedText>Sign in from Home to view group people.</ThemedText>}
      {user && !validID && <ThemedText accessibilityRole="alert">This group link is invalid.</ThemedText>}
      {user && validID && <PeopleList key={`${user.uid}:${groupID}`} user={user} groupID={groupID} />}
    </SafeAreaView>
  );
}

function PeopleList({ user, groupID }: { user: AuthenticatedUser & { uid: string }; groupID: string }) {
  const theme = useTheme();
  const { group } = useGroupDetails(user, groupID);
  const members = useGroupMembers(user, groupID);
  const balances = useGroupBalances(user, groupID);
  const [inviting, setInviting] = useState(false);

  const balanceByUID = new Map(balances.balances.map((balance) => [balance.uid, balance.netAmountMinor]));
  const currency = group?.currency ?? 'EUR';
  const loading = members.loading || balances.loading;
  const error = members.error || balances.error;

  function back() {
    router.dismissTo({ pathname: '/groups/[groupID]', params: { groupID } });
  }

  function retry() {
    if (members.error) members.retry();
    if (balances.error) balances.retry();
  }

  function refresh() {
    members.retry();
    balances.retry();
  }

  function renderMember({ item }: { item: GroupMember }) {
    return (
      <MemberBalanceRow
        member={item}
        isCurrentUser={item.uid === user.uid}
        netAmountMinor={balanceByUID.get(item.uid) ?? 0}
        currency={currency}
      />
    );
  }

  return (
    <>
      <View style={styles.header}>
        <Pressable onPress={back} accessibilityRole="button" accessibilityLabel="Back to group" style={styles.iconButton}>
          <SymbolView name={{ ios: 'arrow.left', android: 'arrow_back', web: 'arrow_back' }} size={24} tintColor={theme.text} />
        </Pressable>
        <ThemedText type="subtitle" style={styles.title} accessibilityRole="header" numberOfLines={1}>
          {group ? `People in ${group.name}` : 'People'}
        </ThemedText>
        <Pressable onPress={() => setInviting(true)} disabled={!group} accessibilityRole="button"
          accessibilityLabel="Invite people" accessibilityState={{ disabled: !group }} style={styles.iconButton}>
          <SymbolView name={{ ios: 'person.badge.plus', android: 'person_add', web: 'person_add' }} size={24} tintColor={theme.text} />
        </Pressable>
      </View>

      {error && (
        <View style={styles.feedback}>
          <ThemedText accessibilityRole="alert">{error}</ThemedText>
          <Button title="Retry" disabled={loading} onPress={retry} />
        </View>
      )}

      {!loading && !error && members.members.length === 1 && (
        <ThemedText>Just you so far. Tap the invite icon above to invite more people.</ThemedText>
      )}

      <FlatList
        data={members.members}
        keyExtractor={(member) => member.uid}
        renderItem={renderMember}
        refreshing={loading}
        onRefresh={refresh}
        contentContainerStyle={styles.list}
      />

      {group && inviting && <InviteUserModal user={user} group={group} onClose={() => setInviting(false)} />}
    </>
  );
}

type MemberBalanceRowProps = {
  member: GroupMember;
  isCurrentUser: boolean;
  netAmountMinor: number;
  currency: string;
};

function MemberBalanceRow({ member, isCurrentUser, netAmountMinor, currency }: MemberBalanceRowProps) {
  const theme = useTheme();
  const amount = formatExpenseAmount(Math.abs(netAmountMinor), currency);

  let label = 'Settled up';
  let color: string = theme.textSecondary;
  if (netAmountMinor > 0) {
    label = isCurrentUser ? 'You are owed' : 'Is owed';
    color = theme.moneyReceivable;
  } else if (netAmountMinor < 0) {
    label = isCurrentUser ? 'You owe' : 'Owes';
    color = theme.moneyOwed;
  }

  return (
    <View style={styles.row}>
      <View style={styles.member}>
        <GroupMemberRow member={member} />
      </View>
      <View style={styles.balance}>
        <ThemedText type="small" style={{ color }}>{label}</ThemedText>
        {netAmountMinor !== 0 && <ThemedText type="smallBold" style={{ color }}>{amount}</ThemedText>}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, paddingHorizontal: 24, gap: 16 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  title: { flex: 1 },
  iconButton: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center' },
  feedback: { gap: 12 },
  list: { paddingBottom: 24 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  member: { flex: 1, minWidth: 0 },
  balance: { alignItems: 'flex-end', maxWidth: '40%' },
});
