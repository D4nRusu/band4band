import { router, useLocalSearchParams } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { useState } from 'react';
import { ActivityIndicator, Alert, Button, Pressable, RefreshControl, ScrollView, StyleSheet, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { AddListItemModal } from '@/components/add-list-item-modal';
import { ErrorMessage } from '@/components/error-message';
import { ListItemRow } from '@/components/list-item-row';
import { ThemedText } from '@/components/themed-text';
import { useAuthSession } from '@/hooks/use-auth-session.android';
import { useGroupMembers } from '@/hooks/use-group-members';
import { useListItems } from '@/hooks/use-list-items';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import type { ListItem } from '@/lib/lists-api';

export default function ListItemsScreen() {
  const theme = useTheme();
  const { groupID, listID, name } = useLocalSearchParams<{ groupID: string; listID: string; name?: string }>();
  const { user, initializing } = useAuthSession();
  const validIDs = typeof groupID === 'string' && groupID.length > 0 && typeof listID === 'string' && listID.length > 0;

  return (
    <SafeAreaView style={[styles.screen, { backgroundColor: theme.background }]}>
      {initializing && <ActivityIndicator accessibilityLabel="Restoring session" />}
      {!initializing && !user && <ThemedText>Sign in from Home to view this list.</ThemedText>}
      {user && !validIDs && <ThemedText accessibilityRole="alert">This list link is invalid.</ThemedText>}
      {user && validIDs && (
        <Items key={`${user.uid}:${groupID}:${listID}`} user={user} groupID={groupID} listID={listID} name={name ?? 'List'} />
      )}
    </SafeAreaView>
  );
}

type ItemsProps = {
  user: AuthenticatedUser & { uid: string };
  groupID: string;
  listID: string;
  name: string;
};

function Items({ user, groupID, listID, name }: ItemsProps) {
  const theme = useTheme();
  const list = useListItems(user, groupID, listID);
  const members = useGroupMembers(user, groupID);
  const [text, setText] = useState('');
  const [adding, setAdding] = useState(false);
  const [showHistory, setShowHistory] = useState(false);

  const memberNameByUID = new Map(members.members.map((member) => [member.uid, member.name || member.username]));
  const openItems = list.items.filter((item) => !item.done);
  const doneItems = sortByDoneAt(list.items.filter((item) => item.done && !item.clearedAt));
  const historyItems = sortByDoneAt(list.items.filter((item) => item.clearedAt));

  function back() {
    router.dismissTo({ pathname: '/groups/[groupID]/lists', params: { groupID } });
  }

  function memberName(uid: string) {
    return memberNameByUID.get(uid) ?? 'Former member';
  }

  async function quickAdd() {
    if (text.trim().length === 0) return;
    if (await list.add({ text, assigneeUID: '', rotationUIDs: [] })) setText('');
  }

  function openAddModal() {
    list.clearSaveError();
    setAdding(true);
  }

  function closeAddModal() {
    setAdding(false);
    setText('');
  }

  function showItemOptions(item: ListItem) {
    if (item.clearedAt) return;

    if (item.createdBy !== user.uid && item.assigneeUID !== user.uid) {
      Alert.alert('Cannot delete item', 'Only the person who added this item or the person assigned to it can delete it.');
      return;
    }

    Alert.alert('Delete item?', `Delete “${item.text}”?`, [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Delete', style: 'destructive', onPress: () => void list.remove(item.id) },
    ]);
  }

  function renderItem(item: ListItem) {
    return (
      <ListItemRow
        key={item.id}
        item={item}
        currentUID={user.uid}
        memberName={memberName}
        disabled={list.saving}
        onToggle={list.toggle}
        onLongPress={showItemOptions}
      />
    );
  }

  return (
    <>
      <View style={styles.header}>
        <Pressable onPress={back} accessibilityRole="button" accessibilityLabel="Back to lists" style={styles.back}>
          <SymbolView name={{ ios: 'arrow.left', android: 'arrow_back', web: 'arrow_back' }} size={24} tintColor={theme.text} />
        </Pressable>
        <ThemedText type="subtitle" style={styles.title} accessibilityRole="header" numberOfLines={1}>{name}</ThemedText>
      </View>

      <ScrollView
        contentContainerStyle={styles.content}
        keyboardShouldPersistTaps="handled"
        refreshControl={<RefreshControl refreshing={list.loading} onRefresh={list.reload} />}
      >
        <View style={styles.addRow}>
          <TextInput
            accessibilityLabel="New item"
            value={text}
            onChangeText={setText}
            onSubmitEditing={quickAdd}
            maxLength={200}
            editable={!list.saving}
            placeholder="Add an item"
            placeholderTextColor={theme.textSecondary}
            style={[styles.input, { color: theme.text, borderColor: theme.textSecondary }]}
          />
          <Button title="Add" onPress={quickAdd} disabled={list.saving || text.trim().length === 0} />
        </View>
        <Button title="Add with assignee or rotation" onPress={openAddModal} disabled={list.saving || members.loading} />

        {!adding && list.saveError && <ErrorMessage message={list.saveError} />}
        {list.error && (
          <View style={styles.feedback}>
            <ErrorMessage message={list.error} />
            <Button title="Retry" onPress={list.reload} disabled={list.loading} />
          </View>
        )}

        {!list.loading && !list.error && openItems.length === 0 && (
          <ThemedText themeColor="textSecondary">Nothing to do. Add an item above.</ThemedText>
        )}
        {openItems.map(renderItem)}

        {doneItems.length > 0 && (
          <View style={styles.section}>
            <View style={styles.sectionHeader}>
              <ThemedText type="smallBold" style={styles.sectionTitle}>Done ({doneItems.length})</ThemedText>
              <Button title="Clear done" onPress={() => void list.clearDone()} disabled={list.saving} />
            </View>
            {doneItems.map(renderItem)}
          </View>
        )}

        {historyItems.length > 0 && (
          <View style={styles.section}>
            <Pressable
              onPress={() => setShowHistory((previous) => !previous)}
              accessibilityRole="button"
              accessibilityState={{ expanded: showHistory }}
              style={styles.sectionHeader}
            >
              <ThemedText type="smallBold" style={styles.sectionTitle}>History ({historyItems.length})</ThemedText>
              <ThemedText type="small" themeColor="textSecondary">{showHistory ? 'Hide' : 'Show'}</ThemedText>
            </Pressable>
            {showHistory && historyItems.map(renderItem)}
          </View>
        )}
      </ScrollView>

      {adding && (
        <AddListItemModal
          currentUID={user.uid}
          members={members.members}
          initialText={text}
          saving={list.saving}
          error={list.saveError}
          onSave={list.add}
          onClose={closeAddModal}
        />
      )}
    </>
  );
}

function sortByDoneAt(items: ListItem[]): ListItem[] {
  return [...items].sort((left, right) => Date.parse(right.doneAt ?? right.createdAt) - Date.parse(left.doneAt ?? left.createdAt));
}

const styles = StyleSheet.create({
  screen: { flex: 1, paddingHorizontal: 24 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingBottom: 8 },
  back: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center' },
  title: { flex: 1 },
  content: { gap: 8, paddingBottom: 24 },
  addRow: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  input: { flex: 1, borderWidth: 1, borderRadius: 8, padding: 12, fontSize: 16 },
  feedback: { gap: 12 },
  section: { gap: 8, marginTop: 16 },
  sectionHeader: { flexDirection: 'row', alignItems: 'center', gap: 12, minHeight: 44 },
  sectionTitle: { flex: 1 },
});
