export function parseEuroAmount(value: string): number {
  const normalized = value.trim().replace(',', '.');
  if (!/^\d+(?:\.\d{1,2})?$/.test(normalized)) {
    throw new Error('Enter the amount using numbers and up to two decimal places, such as 12.50 or 12,50.');
  }

  const [whole, fraction = ''] = normalized.split('.');
  const minor = BigInt(whole) * 100n + BigInt(fraction.padEnd(2, '0'));
  if (minor < 1n) {
    throw new Error('Enter an amount of at least €0.01.');
  }

  if (minor > BigInt(Number.MAX_SAFE_INTEGER)) {
    throw new Error(`This amount is too large. The maximum is ${formatExpenseAmount(Number.MAX_SAFE_INTEGER, 'EUR')}.`);
  }

  return Number(minor);
}

export function formatExpenseAmount(amountMinor: number, currency: string): string {
  const amount = BigInt(amountMinor);
  const whole = amount / 100n;
  const fraction = (amount % 100n).toString().padStart(2, '0');
  return currency === 'EUR' ? `€${whole}.${fraction}` : `${whole}.${fraction} ${currency}`;
}
