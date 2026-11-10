import { SymbolView } from 'expo-symbols';
import { useState } from 'react';
import {
  ActivityIndicator,
  Button,
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  TextInput,
  View,
} from 'react-native';

import { ErrorMessage } from '@/components/error-message';
import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { GroupMember } from '@/lib/groups-api';
import type { ListItemInput } from '@/lib/lists-api';

type AddListItemModalProps = {
  currentUID: string;
  members: GroupMember[];
  initialText: string;
  saving: boolean;
  error: string | null;
  onSave: (input: ListItemInput) => Promise<boolean>;
  onClose: () => void;
};

export function AddListItemModal({ currentUID, members, initialText, saving, error, onSave, onClose }: AddListItemModalProps) {
  const theme = useTheme();
  const [text, setText] = useState(initialText);
  const [assigneeUID, setAssigneeUID] = useState('');
  const [rotate, setRotate] = useState(false);
  const [rotation, setRotation] = useState<string[]>([]);
  const [validationError, setValidationError] = useState<string | null>(null);

  function memberLabel(member: GroupMember) {
    return member.uid === currentUID ? 'You' : member.name || member.username;
  }

  function close() {
    if (!saving) onClose();
  }

  function changeRotate(value: boolean) {
    setRotate(value);
    setRotation(value && assigneeUID ? [assigneeUID] : []);
    setValidationError(null);
  }

  function toggleRotationMember(uid: string) {
    setRotation((previous) => (previous.includes(uid) ? previous.filter((item) => item !== uid) : [...previous, uid]));
    setValidationError(null);
  }

  async function submit() {
    if (text.trim().length === 0) {
      setValidationError('Enter what needs to be done.');
      return;
    }

    if (rotate && rotation.length < 2) {
      setValidationError('Choose at least two people for the rotation.');
      return;
    }

    const input: ListItemInput = rotate
      ? { text, assigneeUID: rotation[0], rotationUIDs: rotation }
      : { text, assigneeUID, rotationUIDs: [] };

    if (await onSave(input)) onClose();
  }

  return (
    <Modal transparent animationType="fade" onRequestClose={close}>
      <KeyboardAvoidingView style={styles.overlay} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <View style={styles.header}>
            <Pressable onPress={close} disabled={saving} accessibilityRole="button" accessibilityLabel="Cancel adding item" style={styles.back}>
              <SymbolView name={{ ios: 'arrow.left', android: 'arrow_back', web: 'arrow_back' }} size={24} tintColor={theme.text} />
            </Pressable>
            <ThemedText type="subtitle" style={styles.title} accessibilityRole="header">Add item</ThemedText>
          </View>

          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <ThemedText>What needs to be done?</ThemedText>
            <TextInput
              accessibilityLabel="Item text"
              value={text}
              onChangeText={(value) => {
                setText(value);
                setValidationError(null);
              }}
              maxLength={200}
              editable={!saving}
              placeholder="Do the dishes"
              placeholderTextColor={theme.textSecondary}
              style={[styles.input, { color: theme.text, borderColor: theme.textSecondary }]}
            />

            <View style={styles.switchRow}>
              <ThemedText style={styles.switchLabel}>Rotate between people</ThemedText>
              <Switch value={rotate} onValueChange={changeRotate} disabled={saving} accessibilityLabel="Rotate between people" />
            </View>

            {rotate ? (
              <>
                <ThemedText type="small" themeColor="textSecondary">
                  Tap people in turn order. The first person goes first; when they tick it, the next person gets it.
                </ThemedText>
                <View style={styles.chips}>
                  {members.map((member) => {
                    const position = rotation.indexOf(member.uid);
                    return (
                      <MemberChip
                        key={member.uid}
                        label={position >= 0 ? `${position + 1}. ${memberLabel(member)}` : memberLabel(member)}
                        selected={position >= 0}
                        disabled={saving}
                        onPress={() => toggleRotationMember(member.uid)}
                      />
                    );
                  })}
                </View>
              </>
            ) : (
              <>
                <ThemedText>Assigned to</ThemedText>
                <View style={styles.chips}>
                  <MemberChip label="Nobody" selected={assigneeUID === ''} disabled={saving} onPress={() => setAssigneeUID('')} />
                  {members.map((member) => (
                    <MemberChip
                      key={member.uid}
                      label={memberLabel(member)}
                      selected={assigneeUID === member.uid}
                      disabled={saving}
                      onPress={() => setAssigneeUID(member.uid)}
                    />
                  ))}
                </View>
              </>
            )}

            {(validationError || error) && <ErrorMessage message={validationError || error || ''} />}
            {saving && <ActivityIndicator accessibilityLabel="Saving item" />}
            <Button title="Add item" onPress={submit} disabled={saving} />
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

type MemberChipProps = {
  label: string;
  selected: boolean;
  disabled: boolean;
  onPress: () => void;
};

function MemberChip({ label, selected, disabled, onPress }: MemberChipProps) {
  const theme = useTheme();

  return (
    <Pressable
      onPress={onPress}
      disabled={disabled}
      accessibilityRole="checkbox"
      accessibilityState={{ checked: selected, disabled }}
      style={({ pressed }) => [
        styles.chip,
        { borderColor: selected ? theme.text : theme.textSecondary },
        selected && { backgroundColor: theme.backgroundElement },
        pressed && styles.pressed,
      ]}
    >
      <ThemedText type={selected ? 'smallBold' : 'small'}>{label}</ThemedText>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  overlay: { flex: 1, justifyContent: 'center', padding: 24, backgroundColor: 'rgba(0, 0, 0, 0.5)' },
  dialog: { width: '100%', maxWidth: 600, maxHeight: '90%', alignSelf: 'center', borderRadius: 16 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 8, padding: 12 },
  back: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center', borderRadius: 24 },
  title: { flex: 1 },
  content: { padding: 24, paddingTop: 8, gap: 16 },
  input: { borderWidth: 1, borderRadius: 8, padding: 12, fontSize: 16 },
  switchRow: { flexDirection: 'row', alignItems: 'center', gap: 12 },
  switchLabel: { flex: 1 },
  chips: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  chip: { minHeight: 40, paddingHorizontal: 14, justifyContent: 'center', borderWidth: 1, borderRadius: 20 },
  pressed: { opacity: 0.7 },
});
