import { StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ThemedText } from '@/components/themed-text';

export default function GroupsScreen() {
  return (
    <SafeAreaView style={styles.screen}>
      <ThemedText>Groups are currently available in the Android development build.</ThemedText>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, justifyContent: 'center', padding: 24 },
});
