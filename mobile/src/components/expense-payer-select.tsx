import { SymbolView } from 'expo-symbols';
import { useState } from 'react';
import { Image, Keyboard, Pressable, ScrollView, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { GroupMember } from '@/lib/groups-api';

type ExpensePayerSelectProps = {
  members: GroupMember[];
  selectedUID: string;
  currentUID: string;
  disabled: boolean;
  onSelect: (uid: string) => void;
};

function memberName(member: GroupMember): string {
  return member.name || member.username || member.uid;
}

export function ExpensePayerSelect({ members, selectedUID, currentUID, disabled, onSelect }: ExpensePayerSelectProps) {
  const theme = useTheme();
  const [open, setOpen] = useState(false);
  const selected = members.find((member) => member.uid === selectedUID);
  const expanded = open && !disabled;

  function toggle() {
    Keyboard.dismiss();
    setOpen((previous) => !previous);
  }

  function choose(uid: string) {
    onSelect(uid);
    setOpen(false);
  }

  return (
    <View style={styles.section}>
      <ThemedText>Paid by</ThemedText>
      <View style={[styles.dropdown, { borderColor: expanded ? theme.textSecondary : theme.backgroundSelected }]}>
        <Pressable onPress={toggle} disabled={disabled} accessibilityRole="button"
          accessibilityLabel={`Paid by ${selected ? memberName(selected) : 'Choose a payer'}`}
          accessibilityHint="Opens the list of group members"
          accessibilityState={{ expanded, disabled }}
          style={({ pressed }) => [styles.trigger, { backgroundColor: pressed ? theme.backgroundSelected : theme.backgroundElement },
            disabled && styles.disabled]}>
          <PayerLabel member={selected} isYou={selectedUID === currentUID} />
          <SymbolView name={expanded
            ? { ios: 'chevron.up', android: 'expand_less', web: 'expand_less' }
            : { ios: 'chevron.down', android: 'expand_more', web: 'expand_more' }} size={20} tintColor={theme.textSecondary} />
        </Pressable>

        {expanded && (
          <ScrollView nestedScrollEnabled keyboardShouldPersistTaps="handled" style={styles.options}>
            {members.map((member) => {
              const isSelected = member.uid === selectedUID;

              return (
                <Pressable key={member.uid} onPress={() => choose(member.uid)} accessibilityRole="radio"
                  accessibilityLabel={`${memberName(member)}${member.uid === currentUID ? ', you' : ''}`}
                  accessibilityState={{ checked: isSelected }}
                  style={({ pressed }) => [styles.option, {
                    backgroundColor: isSelected || pressed ? theme.backgroundSelected : theme.background,
                  }]}>
                  <PayerLabel member={member} isYou={member.uid === currentUID} showDetail />
                  {isSelected && (
                    <SymbolView name={{ ios: 'checkmark', android: 'check', web: 'check' }} size={20} tintColor={theme.text} />
                  )}
                </Pressable>
              );
            })}
          </ScrollView>
        )}
      </View>
    </View>
  );
}

function PayerLabel({ member, isYou, showDetail = false }: { member?: GroupMember; isYou: boolean; showDetail?: boolean }) {
  const theme = useTheme();
  const [failedPhoto, setFailedPhoto] = useState<string | null>(null);
  const name = member ? memberName(member) : 'Choose a payer';
  const initial = Array.from(name.trim())[0]?.toUpperCase() || '?';
  const detail = member?.username ? `@${member.username}` : member?.email;
  const photoURL = member?.photoURL ?? '';
  const showPhoto = photoURL.startsWith('https://') && photoURL !== failedPhoto;

  return (
    <View style={styles.label}>
      {showPhoto ? (
        <Image source={{ uri: photoURL }} style={styles.avatar} accessible={false}
          onError={() => setFailedPhoto(photoURL)} />
      ) : (
        <View style={[styles.avatar, { backgroundColor: theme.backgroundSelected }]}>
          <ThemedText type="smallBold" accessible={false}>{initial}</ThemedText>
        </View>
      )}
      <View style={styles.name}>
        <ThemedText type="smallBold" numberOfLines={1}>{name}{isYou ? ' (You)' : ''}</ThemedText>
        {showDetail && detail && <ThemedText type="small" themeColor="textSecondary" numberOfLines={1}>{detail}</ThemedText>}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: 8 },
  dropdown: { borderWidth: 1, borderRadius: 12, overflow: 'hidden' },
  trigger: { minHeight: 56, padding: 12, flexDirection: 'row', alignItems: 'center', gap: 12 },
  options: { maxHeight: 240 },
  option: { minHeight: 60, padding: 12, flexDirection: 'row', alignItems: 'center', gap: 12 },
  label: { flex: 1, flexDirection: 'row', alignItems: 'center', gap: 12 },
  avatar: { width: 36, height: 36, borderRadius: 18, alignItems: 'center', justifyContent: 'center' },
  name: { flex: 1, gap: 2 },
  disabled: { opacity: 0.5 },
});
