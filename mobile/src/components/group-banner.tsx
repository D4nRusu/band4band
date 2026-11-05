import { Image } from 'expo-image';
import { Pressable, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import type { Group } from '@/lib/groups-api';

type GroupBannerProps = {
  group: Group;
  onBack: () => void;
  onPeoplePress: () => void;
};

export function GroupBanner({ group, onBack, onPeoplePress }: GroupBannerProps) {
  const count = group.memberUIDs.length;
  const peopleLabel = count === 1 ? '1 person' : `${count} people`;

  return (
    <View style={styles.banner}>
      <Image source={require('@/assets/images/group-placeholder.svg')} style={StyleSheet.absoluteFill} contentFit="cover" />
      <View style={styles.shade} pointerEvents="none" />

      <View style={styles.content}>
        <Pressable onPress={onBack} accessibilityRole="button" accessibilityLabel="Back to groups" style={styles.back}>
          <ThemedText style={styles.white}>‹ Groups</ThemedText>
        </Pressable>

        <View style={styles.heading}>
          <ThemedText type="title" style={styles.title} numberOfLines={2}>{group.name}</ThemedText>
          <Pressable onPress={onPeoplePress} accessibilityRole="button"
            accessibilityLabel={`View ${peopleLabel} in this group`} style={styles.people}>
            <ThemedText style={styles.white}>{peopleLabel}</ThemedText>
          </Pressable>
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  banner: { height: 280, width: '100%', backgroundColor: '#24434d' },
  shade: { ...StyleSheet.absoluteFill, backgroundColor: 'rgba(0, 0, 0, 0.45)' },
  content: { flex: 1, padding: 24, justifyContent: 'space-between', alignItems: 'flex-start' },
  back: { paddingVertical: 8, paddingRight: 16 },
  heading: { gap: 12, width: '100%' },
  title: { color: '#ffffff', fontSize: 32, lineHeight: 38 },
  white: { color: '#ffffff' },
  people: { alignSelf: 'flex-start', borderRadius: 20, paddingHorizontal: 16, paddingVertical: 8, backgroundColor: 'rgba(255, 255, 255, 0.18)' },
});
