import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { ActivityIndicator, Button, RefreshControl, ScrollView, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { GroupBanner } from '@/components/group-banner';
import { GroupMembersModal } from '@/components/group-members-modal';
import { ThemedText } from '@/components/themed-text';
import { useAuthSession } from '@/hooks/use-auth-session.android';
import { useGroupDetails } from '@/hooks/use-group-details';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';

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

function GroupDetails({ user, groupID }: { user: AuthenticatedUser; groupID: string }) {
  const { group, loading, error, reload } = useGroupDetails(user, groupID);
  const [showMembers, setShowMembers] = useState(false);

  return (
    <>
      <ScrollView refreshControl={<RefreshControl refreshing={loading} onRefresh={reload} />}>
        {group && (
          <>
            <GroupBanner group={group} onBack={backToGroups} onPeoplePress={() => setShowMembers(true)} />
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

      {group && showMembers && (
        <GroupMembersModal user={user} group={group} onClose={() => setShowMembers(false)} />
      )}
    </>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  feedback: { padding: 24, gap: 16 },
  description: { padding: 24 },
});
