import { Button, StyleSheet, TextInput, View } from 'react-native';

import { ExpensePayerSelect } from '@/components/expense-payer-select';
import { ThemedText } from '@/components/themed-text';
import { useTheme } from '@/hooks/use-theme';
import type { GroupMember } from '@/lib/groups-api';
import { canSplitReceiptItem, type ReceiptReviewItem } from '@/lib/receipt-assignment';

type ReceiptItemEditorProps = {
  item: ReceiptReviewItem;
  members: GroupMember[];
  currentUID: string;
  currency: string;
  disabled: boolean;
  onChange: (patch: Partial<ReceiptReviewItem>) => void;
};

export function ReceiptItemEditor({ item, members, currentUID, currency, disabled, onChange }: ReceiptItemEditorProps) {
  const theme = useTheme();
  const inputStyle = [styles.input, { color: theme.text, borderColor: theme.textSecondary }];
  const availableMembers = members.filter((member) => !item.assignments.some((assignment) => assignment.uid === member.uid));

  function addPerson(uid: string) {
    onChange({ assignments: [...item.assignments, { uid, units: '1' }] });
  }

  function changeUnits(uid: string, units: string) {
    onChange({ assignments: item.assignments.map((assignment) => assignment.uid === uid ? { uid, units } : assignment) });
  }

  function removePerson(uid: string) {
    onChange({ assignments: item.assignments.filter((assignment) => assignment.uid !== uid) });
  }

  return (
    <View style={[styles.card, { borderColor: theme.backgroundSelected }]}>
      <ThemedText type="smallBold">{item.name}</ThemedText>
      {item.quantity !== null && <ThemedText type="small" themeColor="textSecondary">Quantity {item.quantity}</ThemedText>}
      <ThemedText type="small">Line total ({currency})</ThemedText>
      <TextInput value={item.price} onChangeText={(price) => onChange({ price })}
        accessibilityLabel={`Line total for ${item.name}`} style={inputStyle}
        keyboardType="numbers-and-punctuation" editable={!disabled} />

      {canSplitReceiptItem(item) && (
        <Button title={item.splitUnits ? 'Assign all to one person' : 'Split units between people'}
          disabled={disabled} onPress={() => onChange({ splitUnits: !item.splitUnits })} />
      )}

      {!item.splitUnits ? (
        <ExpensePayerSelect label="Assign to" placeholder="Choose a person" members={members}
          currentUID={currentUID} selectedUID={item.assigneeUID} disabled={disabled}
          onSelect={(assigneeUID) => onChange({ assigneeUID })} />
      ) : (
        <View style={styles.assignments}>
          {item.assignments.map((assignment) => {
            const member = members.find((person) => person.uid === assignment.uid);
            const name = member?.name || member?.username || 'Former member';

            return (
              <View key={assignment.uid} style={styles.person}>
                <ThemedText style={styles.name}>{name}</ThemedText>
                <TextInput value={assignment.units} onChangeText={(units) => changeUnits(assignment.uid, units)}
                  accessibilityLabel={`Units for ${name}, ${item.name}`} keyboardType="number-pad"
                  editable={!disabled} style={[...inputStyle, styles.units]} />
                <Button title="Remove" disabled={disabled} onPress={() => removePerson(assignment.uid)} />
              </View>
            );
          })}
          {availableMembers.length > 0 && (
            <ExpensePayerSelect label="Add person" placeholder="Choose a person" members={availableMembers}
              currentUID={currentUID} selectedUID="" disabled={disabled} onSelect={addPerson} />
          )}
          <ThemedText type="small" themeColor="textSecondary">Assign all {item.quantity} units.</ThemedText>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  card: { gap: 10, padding: 14, borderWidth: 1, borderRadius: 12 },
  input: { borderWidth: 1, borderRadius: 8, padding: 10, fontSize: 16 },
  assignments: { gap: 10 },
  person: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  name: { flex: 1 },
  units: { width: 64 },
});
