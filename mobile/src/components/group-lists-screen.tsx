import { router } from 'expo-router';
import { Button, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';

export default function GroupListsScreen() {
  return (
    <View style={{ padding: 24, gap: 16 }}>
      <ThemedText>Group lists are currently available in the Android development build.</ThemedText>
      <Button title="Back to groups" onPress={() => router.replace('/groups')} />
    </View>
  );
}
