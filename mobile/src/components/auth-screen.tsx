import { StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ThemedText } from '@/components/themed-text';

export default function AuthScreen() {
  return (
    <SafeAreaView style={styles.container}>
      <ThemedText>Google Sign-In is currently configured for Android development builds.</ThemedText>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({ container: { flex: 1, justifyContent: 'center', padding: 24 } });
