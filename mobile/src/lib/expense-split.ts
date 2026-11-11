import type { ExpenseShareInput } from '@/lib/expenses-api';
import { formatExpenseAmount, parseEuroAmount } from '@/lib/money';

export function parseUnequalShares(uids: string[], amounts: Record<string, string>): ExpenseShareInput[] {
  return uids.map((uid) => ({
    uid,
    amountMinor: parseEuroAmount(amounts[uid] ?? '', { allowZero: true }),
  }));
}

export function splitRemainder(total: number, shares: ExpenseShareInput[]): bigint {
  return BigInt(total) - shares.reduce((sum, share) => sum + BigInt(share.amountMinor), 0n);
}

export function splitFeedback(remainder: bigint): string {
  if (remainder === 0n) return 'All amounts add up to the expense total.';

  if (remainder > 0n) return `${formatExpenseAmount(remainder, 'EUR')} left to assign.`;

  return `${formatExpenseAmount(-remainder, 'EUR')} over the expense total. Reduce the amounts to match.`;
}
