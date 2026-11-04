import { useState } from 'react';
import { ActivityIndicator, Button, FlatList, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { CreateGroupModal } from '@/components/create-group-modal';
import { GroupCard } from '@/components/group-card';
import { ThemedText } from '@/components/themed-text';
import { useAuthSession } from '@/hooks/use-auth-session.android';
import { useGroups } from '@/hooks/use-groups';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';

export default function GroupsScreen() {
  const theme = useTheme();
  const { user, initializing } = useAuthSession();

  return (
    <SafeAreaView style={[styles.screen, { backgroundColor: theme.background }]}>
      <ThemedText type="title" accessibilityRole="header">Groups</ThemedText>

      {initializing && <ActivityIndicator accessibilityLabel="Restoring session" />}
      {!initializing && !user && <ThemedText>Sign in from Home to see your groups.</ThemedText>}
      {user && <UserGroups key={user.uid} user={user} />}
    </SafeAreaView>
  );
}

function UserGroups({ user }: { user: AuthenticatedUser & { uid: string } }) {
  const {
    groups, loading, error, creating, createError, deletingID,
    submitGroup, removeGroup, reload, clearCreateError,
  } = useGroups(user);
  const [showCreate, setShowCreate] = useState(false);
  const busy = loading || creating || deletingID !== null;

  function openCreate() {
    clearCreateError();
    setShowCreate(true);
  }

  function closeCreate() {
    setShowCreate(false);
    reload();
  }

  return (
    <>
      <Button title="Create a new group" disabled={busy} onPress={openCreate} />

      {error && (
        <View style={styles.feedback}>
          <ThemedText accessibilityRole="alert">{error}</ThemedText>
          <Button title="Retry" disabled={busy} onPress={reload} />
        </View>
      )}

      <FlatList
        data={groups}
        keyExtractor={(group) => group.id}
        refreshing={loading}
        onRefresh={reload}
        contentContainerStyle={styles.list}
        ListEmptyComponent={<GroupsEmptyState loading={loading} hasError={error !== null} />}
        renderItem={({ item }) => (
          <GroupCard group={item} currentUID={user.uid} disabled={busy}
            deleting={deletingID === item.id} onDelete={removeGroup} />
        )}
      />

      {showCreate && (
        <CreateGroupModal creating={creating} error={createError} onCreate={submitGroup} onClose={closeCreate} />
      )}
    </>
  );
}

function GroupsEmptyState({ loading, hasError }: { loading: boolean; hasError: boolean }) {
  if (loading) return <ActivityIndicator accessibilityLabel="Loading groups" />;

  if (hasError) return null;

  return <ThemedText>You aren’t in any groups yet. Create your first group above.</ThemedText>;
}

const styles = StyleSheet.create({
  screen: { flex: 1, paddingHorizontal: 24, gap: 20 },
  feedback: { gap: 12 },
  list: { flexGrow: 1, gap: 12, paddingBottom: 24 },
});
