import { router, useLocalSearchParams } from 'expo-router';
import { SymbolView } from 'expo-symbols';
import { useState } from 'react';
import { ActivityIndicator, Alert, Button, FlatList, Pressable, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { AddCategoryModal } from '@/components/add-category-modal';
import { ThemedText } from '@/components/themed-text';
import { useAuthSession } from '@/hooks/use-auth-session.android';
import { useGroupCategories } from '@/hooks/use-group-categories';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import { categoryIcons, type Category } from '@/lib/categories-api';

export default function GroupCategoriesScreen() {
  const theme = useTheme();
  const { groupID } = useLocalSearchParams<{ groupID: string }>();
  const { user, initializing } = useAuthSession();
  const validID = typeof groupID === 'string' && groupID.length > 0;

  function back() {
    if (validID) router.dismissTo({ pathname: '/groups/[groupID]', params: { groupID } });
    else router.dismissTo('/groups');
  }

  return (
    <SafeAreaView style={[styles.screen, { backgroundColor: theme.background }]}>
      <View style={styles.header}>
        <Pressable onPress={back} accessibilityRole="button" accessibilityLabel="Back to group" style={styles.back}>
          <SymbolView name={{ ios: 'arrow.left', android: 'arrow_back', web: 'arrow_back' }} size={24} tintColor={theme.text} />
        </Pressable>
        <ThemedText type="subtitle" accessibilityRole="header">Categories</ThemedText>
      </View>

      {initializing && <ActivityIndicator accessibilityLabel="Restoring session" />}
      {!initializing && !user && <ThemedText>Sign in from Home to view categories.</ThemedText>}
      {user && !validID && <ThemedText accessibilityRole="alert">This group link is invalid.</ThemedText>}
      {user && validID && <CategoryList key={`${user.uid}:${groupID}`} user={user} groupID={groupID} />}
    </SafeAreaView>
  );
}

function CategoryList({ user, groupID }: { user: AuthenticatedUser; groupID: string }) {
  const theme = useTheme();
  const { categories, loading, error, removingID, remove, saving, saveError, reload, save, clearSaveError } = useGroupCategories(user, groupID);
  const [adding, setAdding] = useState(false);

  const busy = loading || saving || removingID !== null;

  function confirmDelete(category: Category) {
    Alert.alert('Delete category?', `Delete “${category.name}” from this group?`, [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Delete', style: 'destructive', onPress: () => void remove(category.id) },
    ]);
  }

  function openAdd() {
    clearSaveError();
    setAdding(true);
  }

  return (
    <>
      <ThemedText type="small">Shared defaults and custom categories for this group.</ThemedText>
      <Button title="Add new category" disabled={busy || error !== null} onPress={openAdd} />

      {error && (
        <View style={styles.feedback}>
          <ThemedText accessibilityRole="alert">{error}</ThemedText>
          <Button title="Retry" onPress={reload} disabled={busy} />
        </View>
      )}

      <FlatList data={categories} keyExtractor={(category) => category.id} refreshing={loading} onRefresh={reload}
        contentContainerStyle={styles.list}
        renderItem={({ item }) => (
          <View style={[styles.row, { backgroundColor: theme.backgroundElement }]}>
            <ThemedText style={styles.emoji} accessible={false}>{categoryIcons[item.icon].image}</ThemedText>
            <View style={styles.details}>
              <ThemedText>{item.name}</ThemedText>
              <ThemedText type="small" themeColor="textSecondary">{item.isDefault ? 'Default' : 'Custom'}</ThemedText>
            </View>

            {!item.isDefault && (
              <Pressable onPress={() => confirmDelete(item)} disabled={busy}
                accessibilityRole="button" accessibilityLabel={`Delete category ${item.name}`}
                accessibilityState={{ disabled: busy }} style={styles.delete}>
                {removingID === item.id ? (
                  <ActivityIndicator accessibilityLabel="Deleting category" />
                ) : (
                  <SymbolView name={{ ios: 'trash', android: 'delete', web: 'delete' }} size={20} tintColor="#d32f2f" />
                )}
              </Pressable>
            )}
          </View>
        )} />

      {adding && (
        <AddCategoryModal saving={saving} error={saveError} onSave={save} onClose={() => setAdding(false)} />
      )}
    </>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, paddingHorizontal: 24, gap: 16 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  back: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center' },
  feedback: { gap: 12 },
  list: { gap: 8, paddingBottom: 24 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 16, padding: 16, borderRadius: 12 },
  emoji: { fontSize: 28, lineHeight: 36 },
  delete: { width: 44, height: 44, alignItems: 'center', justifyContent: 'center' },
  details: { flex: 1, gap: 4 },
});
