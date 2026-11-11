import { Image } from 'expo-image';
import { SymbolView } from 'expo-symbols';
import { useEffect, useRef, useState } from 'react';
import { ActivityIndicator, Button, KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, StyleSheet, View } from 'react-native';

import { ErrorMessage } from '@/components/error-message';
import { ReceiptAssignmentForm } from '@/components/receipt-assignment-form';
import { ThemedText } from '@/components/themed-text';
import { useReceiptExtraction } from '@/hooks/use-receipt-extraction';
import { useTheme } from '@/hooks/use-theme';
import type { AuthenticatedUser } from '@/lib/api';
import type { Group } from '@/lib/groups-api';
import type { ReceiptExpenseInput } from '@/lib/receipt-assignment';
import { pickReceiptPhoto } from '@/lib/receipt-photo';

type ReceiptScanModalProps = {
  user: AuthenticatedUser & { uid: string };
  group: Group;
  saving: boolean;
  error: string | null;
  onSave: (input: ReceiptExpenseInput) => Promise<boolean>;
  onClose: () => void;
};

export function ReceiptScanModal({ user, group, saving, error, onSave, onClose }: ReceiptScanModalProps) {
  const theme = useTheme();
  const receipt = useReceiptExtraction(user, group.id);
  const [picking, setPicking] = useState(false);
  const active = useRef(true);
  const pickerOpen = useRef(false);

  useEffect(() => {
    active.current = true;
    return () => { active.current = false; };
  }, []);

  async function choosePhoto(source: 'camera' | 'library') {
    if (pickerOpen.current || receipt.scanning || saving) return;

    pickerOpen.current = true;
    setPicking(true);
    receipt.setError(null);
    try {
      const photo = await pickReceiptPhoto(source);
      if (photo && active.current) receipt.choosePhoto(photo);
    } catch (cause) {
      if (active.current) receipt.setError(cause instanceof Error ? cause.message : 'Could not open the photo. Please try again.');
    } finally {
      pickerOpen.current = false;
      if (active.current) setPicking(false);
    }
  }

  function close() {
    if (saving) return;

    receipt.cancel();
    onClose();
  }

  return (
    <Modal transparent animationType="fade" onRequestClose={close}>
      <KeyboardAvoidingView style={styles.overlay} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
        <View style={[styles.dialog, { backgroundColor: theme.background }]} accessibilityViewIsModal>
          <View style={styles.header}>
            <Pressable onPress={close} disabled={saving} accessibilityRole="button" accessibilityLabel="Close receipt scanner" style={styles.back}>
              <SymbolView name={{ ios: 'arrow.left', android: 'arrow_back', web: 'arrow_back' }} size={24} tintColor={theme.text} />
            </Pressable>
            <ThemedText type="subtitle" style={styles.title} accessibilityRole="header">Scan receipt</ThemedText>
          </View>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <ThemedText type="small" themeColor="textSecondary">Take a clear photo of the whole receipt, or select one already on your phone.</ThemedText>
            <View style={styles.actions}>
              <Button title="Take photo" disabled={picking || receipt.scanning || saving} onPress={() => choosePhoto('camera')} />
              <Button title="Choose photo" disabled={picking || receipt.scanning || saving} onPress={() => choosePhoto('library')} />
            </View>
            {picking && <ActivityIndicator accessibilityLabel="Opening receipt photo" />}
            {receipt.photo && (
              <>
                <Image source={{ uri: receipt.photo.uri }} style={styles.photo} contentFit="contain" accessibilityLabel="Selected receipt photo" />
                <Button title={receipt.draft ? 'Scan again' : 'Scan receipt'} onPress={receipt.scan} disabled={picking || receipt.scanning || saving} />
              </>
            )}
            {receipt.scanning && (
              <View style={styles.feedback}>
                <ActivityIndicator accessibilityLabel="Extracting receipt items" />
                <ThemedText type="small">Reading your receipt. This may take a few seconds.</ThemedText>
                <Button title="Cancel scan" onPress={receipt.cancel} />
              </View>
            )}
            {receipt.error && <ErrorMessage message={receipt.error} />}
            {receipt.draft && (
              <ReceiptAssignmentForm user={user} group={group} draft={receipt.draft}
                saving={saving} error={error} onSave={onSave} onSaved={onClose} />
            )}
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  overlay: { flex: 1, justifyContent: 'center', padding: 24, backgroundColor: 'rgba(0, 0, 0, 0.5)' },
  dialog: { width: '100%', maxWidth: 600, maxHeight: '90%', alignSelf: 'center', borderRadius: 16 },
  header: { flexDirection: 'row', alignItems: 'center', gap: 8, padding: 12 },
  back: { width: 48, height: 48, alignItems: 'center', justifyContent: 'center', borderRadius: 24 },
  title: { flex: 1, fontSize: 24, lineHeight: 32 },
  content: { padding: 24, paddingTop: 8, gap: 16 },
  actions: { flexDirection: 'row', flexWrap: 'wrap', gap: 12 },
  photo: { width: '100%', height: 280 },
  feedback: { gap: 12 },
});
