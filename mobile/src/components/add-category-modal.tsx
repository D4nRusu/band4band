import { useState } from 'react';
import {
  ActivityIndicator, Button, KeyboardAvoidingView, Modal, Platform,
  Pressable, ScrollView, StyleSheet, TextInput, View,
} from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import { categoryIcons, type CategoryIcon, type CategoryInput } from '@/lib/categories-api';

type AddCategoryModalProps = {
  saving: boolean;
  error: string | null;
  onSave: (input: CategoryInput) => Promise<boolean>;
  onClose: () => void;
};

export function AddCategoryModal({ saving, error, onSave, onClose }: AddCategoryModalProps) {
  const theme = useTheme();
  const [name, setName] = useState('');
  const [icon, setIcon] = useState<CategoryIcon>('other');
  const [validationError, setValidationError] = useState<string | null>(null);

  function close() {
    if (!saving) onClose();
  }

  async function submit() {
    const normalizedName = name.trim().replace(/\s+/g, ' ');
    const length = Array.from(normalizedName).length;
    if (length < 1 || length > 40) {
      setValidationError('Enter a category name of 1–40 characters.');
      return;
    }

    setValidationError(null);
    if (await onSave({ name: normalizedName, icon })) onClose();
  }

  return (
    <Modal transparent animationType="fade" onRequestClose={close}>
      <KeyboardAvoidingView style={styles.overlay} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <ThemedText type="subtitle" accessibilityRole="header">Add category</ThemedText>
            <TextInput accessibilityLabel="Category name" placeholder="Category name"
              placeholderTextColor={theme.textSecondary} value={name} onChangeText={setName} editable={!saving}
              style={[styles.input, { color: theme.text, borderColor: theme.textSecondary }]} />

            <ThemedText>Choose an icon</ThemedText>
            <View style={styles.icons}>
              {(Object.keys(categoryIcons) as CategoryIcon[]).map((key) => (
                <Pressable key={key} onPress={() => setIcon(key)} disabled={saving} accessibilityRole="button"
                  accessibilityLabel={categoryIcons[key].label} accessibilityState={{ selected: key === icon, disabled: saving }}
                  style={[styles.icon, { borderColor: key === icon ? theme.text : 'transparent', backgroundColor: theme.backgroundElement }]}>
                  <ThemedText style={styles.emoji}>{categoryIcons[key].image}</ThemedText>
                </Pressable>
              ))}
            </View>

            {(validationError || error) && <ThemedText accessibilityRole="alert">{validationError || error}</ThemedText>}
            {saving && <ActivityIndicator accessibilityLabel="Adding category" />}
            <Button title="Add category" disabled={saving} onPress={submit} />
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
  icons: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  icon: { width: 52, height: 52, alignItems: 'center', justifyContent: 'center', borderWidth: 2, borderRadius: 12 },
  emoji: { fontSize: 24, lineHeight: 32 },
});
