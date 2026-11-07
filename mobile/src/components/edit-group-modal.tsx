import { useState } from 'react';
import {
  ActivityIndicator, Button, KeyboardAvoidingView, Modal, Platform,
  ScrollView, StyleSheet, TextInput, View,
} from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import type { Group } from '@/lib/groups-api';
import { useEditGroup } from '@/hooks/use-edit-group';

type EditGroupModalProps = {
  user: AuthenticatedUser;
  group: Group;
  onSaved: (group: Group) => void;
  onClose: () => void;
};

export function EditGroupModal({ user, group, onSaved, onClose }: EditGroupModalProps) {
  const theme = useTheme();
  const { saving, error, save } = useEditGroup(user, group.id);
  const [name, setName] = useState(group.name);
  const [description, setDescription] = useState(group.description);
  const [validationError, setValidationError] = useState<string | null>(null);
  const inputStyle = [styles.input, { color: theme.text, borderColor: theme.textSecondary }];

  async function submit() {
    const input = { name: name.trim(), description: description.trim() };

    if (Array.from(input.name).length < 1 || Array.from(input.name).length > 100) {
      setValidationError('Enter a group name of 1–100 characters.');
      return;
    }

    if (Array.from(input.description).length > 1000) {
      setValidationError('Keep the description within 1,000 characters.');
      return;
    }

    setValidationError(null);

    const updated = await save(input);
    if (updated) onSaved(updated);
  }

  function close() {
    if (!saving) onClose();
  }

  return (
    <Modal transparent animationType="fade" onRequestClose={close}>
      <KeyboardAvoidingView style={styles.overlay} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <ThemedText type="subtitle" accessibilityRole="header">Edit group</ThemedText>

            <Button title="Change image (coming later)" disabled />

            <ThemedText>Group name</ThemedText>
            <TextInput
              accessibilityLabel="Group name"
              value={name}
              onChangeText={(value) => { setName(value); setValidationError(null); }}
              placeholder="e.g. Weekend trip"
              placeholderTextColor={theme.textSecondary}
              style={inputStyle}
              editable={!saving}
            />

            <ThemedText>Description (optional)</ThemedText>
            <TextInput
              accessibilityLabel="Group description"
              value={description}
              onChangeText={(value) => { setDescription(value); setValidationError(null); }}
              placeholder="What is this group for?"
              placeholderTextColor={theme.textSecondary}
              style={[inputStyle, styles.description]}
              multiline
              textAlignVertical="top"
              editable={!saving}
            />

            {(validationError || error) && (
              <ThemedText accessibilityRole="alert">{validationError || error}</ThemedText>
            )}

            {saving && <ActivityIndicator accessibilityLabel="Saving group" />}
            <Button title="Save changes" disabled={saving} onPress={submit} />
            <Button title="Cancel" disabled={saving} onPress={close} />
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: { flex: 1, justifyContent: 'center', padding: 24, backgroundColor: 'rgba(0, 0, 0, 0.5)' },
  dialog: { width: '100%', maxWidth: 500, maxHeight: '90%', alignSelf: 'center', borderRadius: 16 },
  content: { padding: 24, gap: 16 },
  input: { borderWidth: 1, borderRadius: 8, padding: 12, fontSize: 16 },
  description: { minHeight: 100 },
});
