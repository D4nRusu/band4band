import type { ReceiptDraft } from './receipts-api';
import { parseEuroAmount } from './money';

export type ReceiptAssignment = { uid: string; units: string };
export type ReceiptReviewItem = {
  id: string;
  name: string;
  quantity: number | null;
  price: string;
  assigneeUID: string;
  splitUnits: boolean;
  assignments: ReceiptAssignment[];
};

export type ReceiptExpenseItem = {
  amountMinor: number;
  units: number;
  assignments: { uid: string; units: number }[];
};

export type ReceiptExpenseInput = {
  name: string;
  categoryID: string;
  paidBy: string;
  currency: string;
  confirmCurrency: boolean;
  totalMinor: number;
  items: ReceiptExpenseItem[];
};

export function receiptPriceText(amount: number): string {
  const absolute = BigInt(Math.abs(amount));
  return `${amount < 0 ? '-' : ''}${absolute / 100n}.${(absolute % 100n).toString().padStart(2, '0')}`;
}

export function reviewReceipt(draft: ReceiptDraft): ReceiptReviewItem[] {
  return draft.items.map((item) => ({
    id: item.id, name: item.name, quantity: item.quantity ?? null,
    price: receiptPriceText(item.amountMinor), assigneeUID: '', splitUnits: false, assignments: [],
  }));
}

export function parseReceiptPrice(price: string): number {
  const trimmed = price.trim();
  const negative = trimmed.startsWith('-');
  const amount = parseEuroAmount(negative ? trimmed.slice(1) : trimmed, { allowZero: true });
  return negative ? -amount : amount;
}

export function receiptItemsTotal(items: ReceiptReviewItem[]): bigint {
  return items.reduce((sum, item) => sum + BigInt(parseReceiptPrice(item.price)), 0n);
}

export function canSplitReceiptItem(item: ReceiptReviewItem): item is ReceiptReviewItem & { quantity: number } {
  return item.quantity !== null && Number.isSafeInteger(item.quantity) && item.quantity > 1;
}

export function buildReceiptItems(items: ReceiptReviewItem[], memberUIDs: string[]): ReceiptExpenseItem[] {
  return items.map((item) => {
    const amountMinor = parseReceiptPrice(item.price);
    const units = canSplitReceiptItem(item) ? item.quantity : 1;
    const selections = item.splitUnits ? item.assignments : [{ uid: item.assigneeUID, units: String(units) }];
    const assignments = selections.map((selection) => {
      if (!memberUIDs.includes(selection.uid)) {
        throw new Error(`Choose a group member for ${item.name}.`);
      }

      if (!/^[1-9][0-9]*$/.test(selection.units) || !Number.isSafeInteger(Number(selection.units))) {
        throw new Error(`Enter a whole number of units for ${item.name}.`);
      }

      return { uid: selection.uid, units: Number(selection.units) };
    });

    const assigned = assignments.reduce((sum, assignment) => sum + BigInt(assignment.units), 0n);
    const uniqueUIDs = new Set(assignments.map((assignment) => assignment.uid));
    if (assigned !== BigInt(units) || uniqueUIDs.size !== assignments.length) {
      throw new Error(`Assign all ${units} units of ${item.name}, choosing each person once.`);
    }

    return { amountMinor, units, assignments };
  });
}
