import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { ActivityIndicator, Button, RefreshControl, ScrollView, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ExpenseDetailsModal } from '@/components/expense-details-modal';
import { CreateExpenseModal } from '@/components/create-expense-modal';
import { GroupExpenseList } from '@/components/group-expense-list';
import { EditGroupModal } from '@/components/edit-group-modal';
import { GroupBanner } from '@/components/group-banner';
import { GroupSettingsModal } from '@/components/group-settings-modal';
import { SettleUpModal } from '@/components/settle-up-modal';
import { ReceiptScanModal } from '@/components/receipt-scan-modal';
import { ThemedText } from '@/components/themed-text';
import { useAuthSession } from '@/hooks/use-auth-session.android';
import { useGroupExpenses } from '@/hooks/use-group-expenses';
import { useGroupDetails } from '@/hooks/use-group-details';
import { useGroupMembers } from '@/hooks/use-group-members';
import { useGroupSettlement } from '@/hooks/use-group-settlement';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import type { Group } from '@/lib/groups-api';
import type { SettleUpInput } from '@/lib/settlements-api';
import type { Expense } from '@/lib/expenses-api';

function backToGroups() {
  if (router.canGoBack()) router.back();
  else router.replace('/groups');
}

export default function GroupDetailsScreen() {
  const theme = useTheme();
  const { groupID } = useLocalSearchParams<{ groupID: string }>();
  const { user, initializing } = useAuthSession();
  const validGroupID = typeof groupID === 'string' && groupID.length > 0;

  return (
    <SafeAreaView style={[styles.screen, { backgroundColor: theme.background }]}>
      {initializing && <ActivityIndicator accessibilityLabel="Restoring session" />}

      {!initializing && !user && (
        <View style={styles.feedback}>
          <ThemedText>Sign in from Home to view this group.</ThemedText>
          <Button title="Back to groups" onPress={backToGroups} />
        </View>
      )}

      {user && !validGroupID && (
        <View style={styles.feedback}>
          <ThemedText accessibilityRole="alert">This group link is invalid.</ThemedText>
          <Button title="Back to groups" onPress={backToGroups} />
        </View>
      )}

      {user && validGroupID && (
        <GroupDetails key={`${user.uid}:${groupID}`} user={user} groupID={groupID} />
      )}
    </SafeAreaView>
  );
}

function GroupDetails({ user, groupID }: { user: AuthenticatedUser & { uid: string }; groupID: string }) {
  const { group, loading, error, reload, applyUpdate } = useGroupDetails(user, groupID);
  const members = useGroupMembers(user, groupID);
  const expenses = useGroupExpenses(user, groupID);
  const settlement = useGroupSettlement(user, groupID);
  const [menu, setMenu] = useState<'settings' | 'edit' | 'expense' | 'expense-details' | 'settle' | 'receipt' | null>(null);

  const [selectedExpense, setSelectedExpense] = useState<Expense | null>(null);

  function closeMenu() {
    setSelectedExpense(null);
    setMenu(null);
  }

  function refreshAll() {
    reload();
    members.retry();
    expenses.reload();
    settlement.reload();
  }

  function showExpenseDetails(expense: Expense) {
    setSelectedExpense(expense);
    setMenu('expense-details');
  }

  function deletedExpense() {
    if (selectedExpense) expenses.removeFromList(selectedExpense.id);
    closeMenu();
  }

  function openExpense() {
    expenses.clearSaveError();
    setMenu('expense');
  }

  function openSettle() {
    settlement.clearSettleError();
    setMenu('settle');
  }

  function openReceipt() {
    expenses.clearSaveError();
    setMenu('receipt');
  }

  function closeExpense() {
    closeMenu();
    expenses.reload();
  }

  function openPeople() {
    router.push({ pathname: '/groups/[groupID]/people', params: { groupID } });
  }

  function openStats() {
    router.push({ pathname: '/groups/[groupID]/stats', params: { groupID } });
  }

  function openLists() {
    router.push({ pathname: '/groups/[groupID]/lists', params: { groupID } });
  }

  function openCategories() {
    closeMenu();
    router.push({ pathname: '/groups/[groupID]/categories', params: { groupID } });
  }

  function savedGroup(updated: Group) {
    applyUpdate(updated);
    setMenu('settings');
  }

  const memberNameByUID = new Map(members.members.map((member) => [member.uid, member.name]));
  const owedDebts = (settlement.owedSummary?.debts ?? []).map((debt) => ({
    toUID: debt.toUID,
    toName: memberNameByUID.get(debt.toUID) ?? 'Unknown member',
    amountMinor: debt.amountMinor,
  }));

  const receivableDebts = (settlement.receivableSummary?.debts ?? []).map((debt) => ({
    fromUID: debt.fromUID,
    fromName: memberNameByUID.get(debt.fromUID) ?? 'Unknown member',
    amountMinor: debt.amountMinor,
  }));

  async function submitSettlement(input: SettleUpInput) {
    const settled = await settlement.settle(input);
    if (settled) expenses.reload();

    return settled;
  }

  return (
    <>
      <ScrollView refreshControl={<RefreshControl refreshing={loading || expenses.loading || settlement.loading} onRefresh={refreshAll} />}>
        {group && (
          <>
            <GroupBanner group={group} onBack={backToGroups}
              onPeoplePress={openPeople} onSettingsPress={() => setMenu('settings')} />
            <GroupExpenseList
              currentUID={user.uid}
              expenses={expenses.expenses}
              settlements={settlement.settlements}
                debts={owedDebts}
                receivables={receivableDebts}
                debtCurrency={settlement.owedSummary?.currency ?? settlement.receivableSummary?.currency ?? group.currency}
              debtsLoading={settlement.loading}
              debtsError={settlement.error}
              loading={expenses.loading}
              error={expenses.error}
              saving={expenses.saving}
              settleDisabled={expenses.saving || settlement.settling}
              onAdd={openExpense}
              onSettleUp={openSettle}
              onStats={openStats}
              onLists={openLists}
              onScanReceipt={openReceipt}
              onOpen={showExpenseDetails}
              onRetry={expenses.reload}
              onRetryDebts={settlement.reload}
            />
          </>
        )}

        {!group && (
          <View style={styles.feedback}>
            <Button title="Back to groups" onPress={backToGroups} />
            {loading && <ActivityIndicator accessibilityLabel="Loading group" />}
          </View>
        )}

        {error && (
          <View style={styles.feedback}>
            <ThemedText accessibilityRole="alert">{error}</ThemedText>
            <Button title="Retry" disabled={loading} onPress={reload} />
          </View>
        )}
      </ScrollView>

      {group && selectedExpense && menu === 'expense-details' && (
        <ExpenseDetailsModal key={selectedExpense.id} user={user} expense={selectedExpense}
          onClose={closeMenu} onDeleted={deletedExpense} />
      )}

      {group && menu === 'expense' && (
        <CreateExpenseModal user={user} group={group} saving={expenses.saving} error={expenses.saveError}
          onSave={expenses.save} onClose={closeExpense} />
      )}

      {group && menu === 'receipt' && (
        <ReceiptScanModal user={user} group={group} saving={expenses.saving} error={expenses.saveError}
          onSave={expenses.save} onClose={closeExpense} />
      )}

      {group && menu === 'settle' && (
        <SettleUpModal
          currentUID={user.uid}
          currency={group.currency}
          members={members.members}
          membersLoading={members.loading}
          membersError={members.error}
          debts={owedDebts}
          receivables={receivableDebts}
          saving={settlement.settling}
          error={settlement.settleError}
          onRetryMembers={members.retry}
          onSettle={submitSettlement}
          onClose={closeMenu}
        />
      )}

      {group && menu === 'edit' && (
        <EditGroupModal user={user} group={group} onSaved={savedGroup} onClose={() => setMenu('settings')} />
      )}

      {group && menu === 'settings' && (
        <GroupSettingsModal user={user} group={group} onClose={closeMenu} onEdit={() => setMenu('edit')} onCategories={openCategories} />
      )}
    </>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  feedback: { padding: 24, gap: 16 },
});
