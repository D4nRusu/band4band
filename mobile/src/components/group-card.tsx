import { Image } from 'expo-image';
import { Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { Group } from '@/lib/groups-api';

type GroupCardProps = {
  group: Group;
  disabled: boolean;
  onOpen: (group: Group) => void;
};

export function GroupCard({ group, disabled, onOpen }: GroupCardProps) {
  const theme = useTheme();

  return (
    <Pressable
      onPress={() => onOpen(group)}
      disabled={disabled}
      accessibilityRole="link"
      accessibilityLabel={`Open group ${group.name}`}
      accessibilityState={{ disabled }}
      style={({ pressed }) => [
        styles.card,
        { backgroundColor: theme.backgroundElement },
        pressed && styles.pressed,
      ]}>
      <Image
        source={require('@/assets/images/group-placeholder.svg')}
        style={styles.image}
        contentFit="cover"
        accessible={false}
      />

      <View style={styles.details}>
        <ThemedText type="subtitle" style={styles.name} numberOfLines={2}>{group.name}</ThemedText>
        {group.description !== '' && (
          <ThemedText type="small" themeColor="textSecondary" numberOfLines={2}>
            {group.description}
          </ThemedText>
        )}
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: { borderRadius: 16, overflow: 'hidden' },
  image: { width: '100%', aspectRatio: 2.5, backgroundColor: '#24434d' },
  details: { padding: 20, gap: 8 },
  name: { fontSize: 22, lineHeight: 28 },
  pressed: { opacity: 0.8 },
});
