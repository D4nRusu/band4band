import assert from 'node:assert/strict';
import { test } from 'node:test';

import { parseEuroAmount, formatExpenseAmount } from '../src/lib/money.ts';

test('converts EUR input to exact cents without rounding', () => {
  for (const [input, expected] of [
    ['12.50', 1250], ['12,50', 1250], ['0.29', 29], ['0.01', 1],
    ['1', 100], ['1.2', 120], [' 001.02 ', 102], ['90071992547409.91', Number.MAX_SAFE_INTEGER],
  ]) assert.equal(parseEuroAmount(input), expected);
});

test('rejects zero, negative, ambiguous, overprecise, and unsafe values', () => {
  for (const input of ['', '0', '-1', '0.001', '1e3', '1,000.00', 'NaN', 'Infinity', '1.', '90071992547409.92']) {
    assert.throws(() => parseEuroAmount(input));
  }
});

test('formats exact cents, including the largest safe JSON integer', () => {
  assert.equal(formatExpenseAmount(1, 'EUR'), '€0.01');
  assert.equal(formatExpenseAmount(1250, 'EUR'), '€12.50');
  assert.equal(formatExpenseAmount(Number.MAX_SAFE_INTEGER, 'EUR'), '€90071992547409.91');
});

test('allows explicit zero contributions without allowing a zero expense total', () => {
  for (const value of ['0', '0.00', '0,00']) {
    assert.equal(parseEuroAmount(value, { allowZero: true }), 0);
    assert.throws(() => parseEuroAmount(value));
  }

  for (const value of ['', '-1', '0.001', '90071992547409.92']) {
    assert.throws(() => parseEuroAmount(value, { allowZero: true }));
  }

  assert.equal(formatExpenseAmount(BigInt(Number.MAX_SAFE_INTEGER) * 2n, 'EUR'), '€180143985094819.82');
});
