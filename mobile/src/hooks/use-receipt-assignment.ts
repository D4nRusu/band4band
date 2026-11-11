import { useState } from 'react';

import { useExpenseOptions } from '@/hooks/use-expense-options';
import type { AuthenticatedUser } from '@/lib/api';
import type { Group } from '@/lib/groups-api';
import { parseEuroAmount } from '@/lib/money';
import {
  buildReceiptItems, receiptItemsTotal, receiptPriceText, reviewReceipt,
  type ReceiptExpenseInput, type ReceiptReviewItem,
} from '@/lib/receipt-assignment';
import type { ReceiptDraft } from '@/lib/receipts-api';

export function useReceiptAssignment(user: AuthenticatedUser & { uid: string }, group: Group, draft: ReceiptDraft) {
  const options = useExpenseOptions(user, group.id);
  const [items, setItems] = useState(() => reviewReceipt(draft));
  const [details, setDetails] = useState({
    name: 'Receipt', total: receiptPriceText(draft.totalMinor),
    categoryID: 'default_other', paidBy: user.uid, confirmCurrency: false,
  });
  const [validationError, setValidationError] = useState<string | null>(null);

  function changeDetails(patch: Partial<typeof details>) {
    setDetails((previous) => ({ ...previous, ...patch }));
    setValidationError(null);
  }

  function changeItem(id: string, patch: Partial<ReceiptReviewItem>) {
    setItems((previous) => previous.map((item) => item.id === id ? { ...item, ...patch } : item));
    setValidationError(null);
  }

  function buildExpense(): ReceiptExpenseInput {
    const name = details.name.trim();
    if (Array.from(name).length < 1 || Array.from(name).length > 100) {
      throw new Error('Give the expense a name between 1 and 100 characters.');
    }

    if (!options.categories.some((category) => category.id === details.categoryID)) {
      throw new Error('Choose a category from the current list.');
    }

    if (!options.members.some((member) => member.uid === details.paidBy)) {
      throw new Error('Choose a payer from the current group members.');
    }

    if (draft.currency !== group.currency && !details.confirmCurrency) {
      throw new Error(`Confirm using the reviewed amounts in ${group.currency} before saving.`);
    }

    const totalMinor = parseEuroAmount(details.total);
    if (receiptItemsTotal(items) !== BigInt(totalMinor)) {
      throw new Error('The item prices and receipt total differ. Correct the line totals or the receipt total before saving.');
    }

    return {
      name, categoryID: details.categoryID, paidBy: details.paidBy,
      currency: draft.currency, confirmCurrency: details.confirmCurrency, totalMinor,
      items: buildReceiptItems(items, options.members.map((member) => member.uid)),
    };
  }

  let total: bigint | null = null;
  let totalsMatch: boolean | null = null;
  try {
    total = receiptItemsTotal(items);
    totalsMatch = total === BigInt(parseEuroAmount(details.total));
  } catch {
    // Incomplete price input is validated when the user confirms.
  }

  return { options, items, details, validationError, setValidationError, changeDetails, changeItem, buildExpense, total, totalsMatch };
}
