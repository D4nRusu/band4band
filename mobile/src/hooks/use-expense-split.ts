import { useState } from 'react';

import { parseUnequalShares, splitFeedback, splitRemainder } from '@/lib/expense-split';
import type { ExpenseShareInput, ExpenseSplit } from '@/lib/expenses-api';
import type { GroupMember } from '@/lib/groups-api';
import { parseEuroAmount } from '@/lib/money';

export function useExpenseSplit(members: GroupMember[], amount: string) {
  const [mode, setMode] = useState<'equal' | 'unequal'>('equal');
  const [excludedUIDs, setExcludedUIDs] = useState<string[]>([]);
  const [amounts, setAmounts] = useState<Record<string, string>>({});
  const participantUIDs = members.filter((member) => !excludedUIDs.includes(member.uid)).map((member) => member.uid);

  function toggleParticipant(uid: string) {
    setExcludedUIDs((previous) => previous.includes(uid)
      ? previous.filter((excluded) => excluded !== uid)
      : [...previous, uid]);
  }

  function changeShare(uid: string, value: string) {
    setAmounts((previous) => ({ ...previous, [uid]: value }));
  }

  function buildSplit(total: number): ExpenseSplit {
    if (participantUIDs.length === 0) {
      throw new Error('Nobody is included in the split. Select at least one contributor.');
    }

    if (mode === 'equal') return { splitType: 'equal', participantUIDs };

    let contributions: ExpenseShareInput[];
    try {
      contributions = parseUnequalShares(participantUIDs, amounts);
    } catch {
      throw new Error('Enter an amount for every selected contributor, using up to two decimal places. Enter 0 for a zero share.');
    }

    const remainder = splitRemainder(total, contributions);
    if (remainder !== 0n) throw new Error(splitFeedback(remainder));

    return { splitType: 'unequal', contributions };
  }

  let feedback: string | null = null;
  if (mode === 'unequal') {
    try {
      const total = parseEuroAmount(amount);
      feedback = splitFeedback(splitRemainder(total, parseUnequalShares(participantUIDs, amounts)));
    } catch {
      feedback = 'Enter the expense total and an amount for each selected contributor.';
    }
  }

  return { mode, setMode, amounts, participantUIDs, toggleParticipant, changeShare, buildSplit, feedback };
}
