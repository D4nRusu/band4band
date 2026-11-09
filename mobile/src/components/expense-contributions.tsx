import { ActivityIndicator, Button, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useGroupMembers } from '@/hooks/use-group-members';
import type { AuthenticatedUser } from '@/lib/api';

type ExpenseContributionsProps = {
  user: AuthenticatedUser;
  groupID: string;
};

export function ExpenseContributions({ user, groupID }: ExpenseContributionsProps) {
  const { members, loading, error, retry } = useGroupMembers(user, groupID);

  return (
    <View style={styles.section}>
      <ThemedText type="smallBold" accessibilityRole="header">Contributions</ThemedText>
      <ThemedText type="small" themeColor="textSecondary">
        Contributions have not been set. These are the current group members; participants and shares will be assigned later.
      </ThemedText>

      {loading && <ActivityIndicator accessibilityLabel="Loading group members" />}
      {error && (
        <View style={styles.section}>
          <ThemedText accessibilityRole="alert">{error}</ThemedText>
          <Button title="Retry members" onPress={retry} disabled={loading} />
        </View>
      )}

      {members.map((member) => (
        <View key={member.uid} style={styles.row}>
          <ThemedText style={styles.name}>{member.name || member.username || 'Member'}</ThemedText>
          <ThemedText type="small" themeColor="textSecondary">Not set</ThemedText>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: 12 },
  row: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingVertical: 8 },
  name: { flex: 1 },
});
