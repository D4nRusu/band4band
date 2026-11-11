import assert from 'node:assert/strict';
import { registerHooks } from 'node:module';
import { test } from 'node:test';

// Node needs the file extension for Metro's local TypeScript import.
const hooks = registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === './money' && context.parentURL?.endsWith('/src/lib/receipt-assignment.ts')) {
      return nextResolve('./money.ts', context);
    }
    return nextResolve(specifier, context);
  },
});
const {
  buildReceiptItems, canSplitReceiptItem, parseReceiptPrice, receiptItemsTotal, receiptPriceText, reviewReceipt,
} = await import('../src/lib/receipt-assignment.ts');
hooks.deregister();

function item(quantity = 3) {
  return reviewReceipt({ items: [{ id: 'line', name: 'Cables', quantity, amountMinor: 1996 }] })[0];
}

test('assigns the full line or whole units without multiplying the line price again', () => {
  const row = item();
  row.assigneeUID = 'a';
  assert.deepEqual(buildReceiptItems([row], ['a', 'b']), [{
    amountMinor: 1996, units: 3, assignments: [{ uid: 'a', units: 3 }],
  }]);

  row.splitUnits = true;
  row.assignments = [{ uid: 'a', units: '2' }, { uid: 'b', units: '1' }];
  assert.deepEqual(buildReceiptItems([row], ['a', 'b'])[0].assignments, [
    { uid: 'a', units: 2 }, { uid: 'b', units: 1 },
  ]);
  assert.equal(receiptItemsTotal([row]), 1996n);
});

test('requires every unit and a current member, and rejects duplicate or fractional selections', () => {
  for (const assignments of [
    [], [{ uid: 'a', units: '2' }], [{ uid: 'a', units: '4' }],
    [{ uid: 'a', units: '1.5' }, { uid: 'b', units: '1.5' }],
    [{ uid: 'a', units: '1' }, { uid: 'a', units: '2' }],
    [{ uid: 'former-member', units: '3' }],
  ]) {
    const row = { ...item(), splitUnits: true, assignments };
    assert.throws(() => buildReceiptItems([row], ['a', 'b']));
  }
  assert.throws(() => buildReceiptItems([item()], ['a', 'b']));
});

test('fractional and unknown quantities are assigned as one full line', () => {
  for (const quantity of [0.75, null, undefined, 1]) {
    const row = { ...item(), quantity: quantity ?? null, assigneeUID: 'a' };
    assert.equal(canSplitReceiptItem(row), false);
    assert.equal(buildReceiptItems([row], ['a'])[0].units, 1);
  }
});

test('corrected prices and discount lines retain exact cents', () => {
  const rows = [{ ...item(), price: '20,01' }, { ...item(1), price: '-0.05' }];
  assert.equal(receiptItemsTotal(rows), 1996n);
  assert.equal(parseReceiptPrice('-0.05'), -5);
  for (const value of [0, 1996, -5, Number.MAX_SAFE_INTEGER]) {
    assert.equal(parseReceiptPrice(receiptPriceText(value)), value);
  }
  for (const price of ['1.001', '--1', 'Infinity', '1e3', '90071992547409.92']) {
    assert.throws(() => parseReceiptPrice(price));
  }
});
