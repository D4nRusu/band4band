import { StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';

export function ErrorMessage({ message }: { message: string }) {
  const theme = useTheme();

  return (
    <View accessibilityRole="alert" accessibilityLiveRegion="polite"
      style={[styles.container, { backgroundColor: theme.errorBackground, borderColor: theme.errorBorder }]}>
      <ThemedText style={[styles.message, { color: theme.error }]}>{message}</ThemedText>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { padding: 12, borderWidth: 1, borderRadius: 10 },
  message: { fontSize: 16, lineHeight: 24 },
});
