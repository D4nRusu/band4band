import { router } from 'expo-router';
import { useState } from 'react';
import { ActivityIndicator, Button, FlatList, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { CreateGroupModal } from '@/components/create-group-modal';
import { GroupInvitations } from '@/components/group-invitations';
import { InviteUserModal } from '@/components/invite-user-modal';
import { GroupCard } from '@/components/group-card';
import { ThemedText } from '@/components/themed-text';
import { useAuthSession } from '@/hooks/use-auth-session.android';
import { useGroupInvitations } from '@/hooks/use-group-invitations';
import { useGroups } from '@/hooks/use-groups';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import type { Group } from '@/lib/groups-api';

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
    groups, loading, error, creating, createError, removingID,
    submitGroup, removeGroup, leave, reload, clearCreateError,
  } = useGroups(user);
  const inbox = useGroupInvitations(user, reload);
  const [showCreate, setShowCreate] = useState(false);
  const [inviteGroup, setInviteGroup] = useState<Group | null>(null);
  const busy = loading || creating || removingID !== null || inbox.respondingID !== null;

  function openCreate() {
    clearCreateError();
    setShowCreate(true);
  }

  function closeCreate() {
    setShowCreate(false);
    reload();
  }

  function refreshAll() {
    reload();
    inbox.reload();
  }

  function openGroup(group: Group) {
    router.push({ pathname: '/groups/[groupID]', params: { groupID: group.id } });
  }

  return (
    <>
      <Button title="Create a new group" disabled={busy} onPress={openCreate} />

      {error && (
        <View style={styles.feedback}>
          <ThemedText accessibilityRole="alert">{error}</ThemedText>
          <Button title="Retry" disabled={busy} onPress={refreshAll} />
        </View>
      )}

      <FlatList
        data={groups}
        keyExtractor={(group) => group.id}
        refreshing={loading || inbox.loading}
        onRefresh={refreshAll}
        contentContainerStyle={styles.list}
        ListHeaderComponent={
          <GroupInvitations
            invitations={inbox.invitations}
            loading={inbox.loading}
            error={inbox.error}
            respondingID={inbox.respondingID}
            disabled={busy}
            onRespond={inbox.respond}
          />
        }
        ListEmptyComponent={<GroupsEmptyState loading={loading} hasError={error !== null} />}
        renderItem={({ item }) => (
          <GroupCard
            group={item}
            currentUID={user.uid}
            disabled={busy}
            removing={removingID === item.id}
            onDelete={removeGroup}
            onLeave={leave}
            onInvite={setInviteGroup}
            onOpen={openGroup}
          />
        )}
      />

      {showCreate && (
        <CreateGroupModal creating={creating} error={createError} onCreate={submitGroup} onClose={closeCreate} />
      )}

      {inviteGroup && (
        <InviteUserModal user={user} group={inviteGroup} onClose={() => setInviteGroup(null)} />
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
