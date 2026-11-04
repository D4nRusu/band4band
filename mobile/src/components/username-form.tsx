import { Button, StyleSheet, TextInput } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';

type UsernameFormProps = {
  username: string;
  error: string | null;
  disabled: boolean;
  onChange: (value: string) => void;
  onSubmit: () => Promise<void>;
};

export function UsernameForm({ username, error, disabled, onChange, onSubmit }: UsernameFormProps) {
  const theme = useTheme();

  function submit() {
    if (!disabled) void onSubmit();
  }

  return (
    <>
      <ThemedText>Choose a unique username so friends can find you.</ThemedText>
      <TextInput
        accessibilityLabel="Username"
        value={username}
        onChangeText={onChange}
        autoCapitalize="none"
        autoCorrect={false}
        maxLength={30}
        editable={!disabled}
        placeholder="Username"
        placeholderTextColor={theme.textSecondary}
        style={[styles.input, { color: theme.text, borderColor: theme.textSecondary }]}
        onSubmitEditing={submit}
      />
      <ThemedText type="small">3-30 letters, numbers, or underscores. Saved in lowercase.</ThemedText>
      {error && <ThemedText accessibilityRole="alert">{error}</ThemedText>}
      <Button title="Save username" disabled={disabled} onPress={submit} />
    </>
  );
}

const styles = StyleSheet.create({
  input: { width: '100%', borderWidth: 1, borderRadius: 8, padding: 12, fontSize: 16 },
});
