import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { ActivityIndicator, Button, RefreshControl, ScrollView, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { EditGroupModal } from '@/components/edit-group-modal';
import { GroupBanner } from '@/components/group-banner';
import { GroupMembersModal } from '@/components/group-members-modal';
import { GroupSettingsModal } from '@/components/group-settings-modal';
import { InviteUserModal } from '@/components/invite-user-modal';
import { ThemedText } from '@/components/themed-text';
import { useAuthSession } from '@/hooks/use-auth-session.android';
import { useGroupDetails } from '@/hooks/use-group-details';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import type { Group } from '@/lib/groups-api';

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
  const [menu, setMenu] = useState<'members' | 'invite' | 'settings' | 'edit' | null>(null);

  function closeMenu() {
    setMenu(null);
  }

  function openCategories() {
    closeMenu();
    router.push({ pathname: '/groups/[groupID]/categories', params: { groupID } });
  }

  function savedGroup(updated: Group) {
    applyUpdate(updated);
    setMenu('settings');
  }

  return (
    <>
      <ScrollView refreshControl={<RefreshControl refreshing={loading} onRefresh={reload} />}>
        {group && (
          <>
            <GroupBanner group={group} onBack={backToGroups}
              onPeoplePress={() => setMenu('members')} onSettingsPress={() => setMenu('settings')} />
            {group.description !== '' && <ThemedText style={styles.description}>{group.description}</ThemedText>}
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

      {group && menu === 'members' && (
        <GroupMembersModal user={user} group={group} onClose={closeMenu} onInvite={() => setMenu('invite')} />
      )}

      {group && menu === 'invite' && (
        <InviteUserModal user={user} group={group} onClose={() => setMenu('members')} />
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
  description: { padding: 24 },
});
