import { router, useLocalSearchParams } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { useState } from 'react';
import { ActivityIndicator, Alert, Button, FlatList, Pressable, StyleSheet, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ErrorMessage } from '@/components/error-message';
import { ThemedText } from '@/components/themed-text';
import { useAuthSession } from '@/hooks/use-auth-session.android';
import { useGroupLists } from '@/hooks/use-group-lists';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import type { GroupList } from '@/lib/lists-api';

export default function GroupListsScreen() {
  const theme = useTheme();
  const { groupID } = useLocalSearchParams<{ groupID: string }>();
  const { user, initializing } = useAuthSession();
  const validID = typeof groupID === 'string' && groupID.length > 0;

  return (
    <SafeAreaView style={[styles.screen, { backgroundColor: theme.background }]}>
      {initializing && <ActivityIndicator accessibilityLabel="Restoring session" />}
      {!initializing && !user && <ThemedText>Sign in from Home to view group lists.</ThemedText>}
      {user && !validID && <ThemedText accessibilityRole="alert">This group link is invalid.</ThemedText>}
      {user && validID && <Lists key={`${user.uid}:${groupID}`} user={user} groupID={groupID} />}
    </SafeAreaView>
  );
}

function Lists({ user, groupID }: { user: AuthenticatedUser & { uid: string }; groupID: string }) {
  const theme = useTheme();
  const { lists, loading, error, saving, saveError, reload, create, remove } = useGroupLists(user, groupID);
  const [name, setName] = useState('');

  function back() {
    router.dismissTo({ pathname: '/groups/[groupID]', params: { groupID } });
  }

  async function submit() {
    if (name.trim().length === 0) return;
    if (await create(name)) setName('');
  }

  function open(list: GroupList) {
    router.push({ pathname: '/groups/[groupID]/lists/[listID]', params: { groupID, listID: list.id, name: list.name } });
  }

  function confirmDelete(list: GroupList) {
    if (list.createdBy !== user.uid) {
      Alert.alert('Cannot delete list', 'Only the person who created this list can delete it.');
      return;
    }

    Alert.alert('Delete list?', `Delete “${list.name}” and all of its items? This cannot be undone.`, [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Delete', style: 'destructive', onPress: () => void remove(list.id) },
    ]);
  }

  function renderList({ item }: { item: GroupList }) {
    return (
      <Pressable
        onPress={() => open(item)}
        onLongPress={() => confirmDelete(item)}
        disabled={saving}
        accessibilityRole="button"
        accessibilityLabel={`Open list ${item.name}`}
        accessibilityHint="Long press to delete"
        style={({ pressed }) => [styles.row, { backgroundColor: theme.backgroundElement }, pressed && styles.pressed]}
      >
        <ThemedText style={styles.rowText}>{item.name}</ThemedText>
        <SymbolView name={{ ios: 'chevron.right', android: 'chevron_right', web: 'chevron_right' }} size={20} tintColor={theme.textSecondary} />
      </Pressable>
    );
  }

  return (
    <>
      <View style={styles.header}>
        <Pressable onPress={back} accessibilityRole="button" accessibilityLabel="Back to group" style={styles.back}>
          <SymbolView name={{ ios: 'arrow.left', android: 'arrow_back', web: 'arrow_back' }} size={24} tintColor={theme.text} />
        </Pressable>
        <ThemedText type="subtitle" accessibilityRole="header">Lists</ThemedText>
      </View>

      <View style={styles.createRow}>
        <TextInput
          accessibilityLabel="New list name"
          value={name}
          onChangeText={setName}
          onSubmitEditing={submit}
          maxLength={50}
          editable={!saving}
          placeholder="New list, e.g. Shopping"
          placeholderTextColor={theme.textSecondary}
          style={[styles.input, { color: theme.text, borderColor: theme.textSecondary }]}
        />
        <Button title="Create" onPress={submit} disabled={saving || name.trim().length === 0} />
      </View>
      {saveError && <ErrorMessage message={saveError} />}

      {error && (
        <View style={styles.feedback}>
          <ErrorMessage message={error} />
          <Button title="Retry" onPress={reload} disabled={loading} />
        </View>
      )}

      {!loading && !error && lists.length === 0 && (
        <ThemedText themeColor="textSecondary">No lists yet. Create one for shopping, chores, or anything the group shares.</ThemedText>
      )}

      <FlatList
        data={lists}
        keyExtractor={(list) => list.id}
        renderItem={renderList}
        refreshing={loading}
        onRefresh={reload}
        contentContainerStyle={styles.list}
      />
    </>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, paddingHorizontal: 24, gap: 16 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  back: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center' },
  createRow: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  input: { flex: 1, borderWidth: 1, borderRadius: 8, padding: 12, fontSize: 16 },
  feedback: { gap: 12 },
  list: { gap: 8, paddingBottom: 24 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 12, padding: 16, borderRadius: 12 },
  rowText: { flex: 1 },
  pressed: { opacity: 0.7 },
});
