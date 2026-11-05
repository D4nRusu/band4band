import { router } from 'expo-router';
import { Button, StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ThemedText } from '@/components/themed-text';

export default function GroupDetailsScreen() {
  return (
    <SafeAreaView style={styles.screen}>
      <ThemedText>Group pages are currently available in the Android development build.</ThemedText>
      <Button title="Back to groups" onPress={() => router.replace('/groups')} />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, justifyContent: 'center', padding: 24, gap: 16 },
});
