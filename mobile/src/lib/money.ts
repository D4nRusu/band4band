export function parseEuroAmount(value: string): number {
  const normalized = value.trim().replace(',', '.');
  if (!/^\d+(?:\.\d{1,2})?$/.test(normalized)) {
    throw new Error('Enter a positive EUR amount with at most two decimal places, for example 12.50.');
  }

  const [whole, fraction = ''] = normalized.split('.');
  const minor = BigInt(whole) * 100n + BigInt(fraction.padEnd(2, '0'));
  if (minor < 1n || minor > BigInt(Number.MAX_SAFE_INTEGER)) {
    throw new Error('The amount must be positive and within the supported range.');
  }

  return Number(minor);
}

export function formatExpenseAmount(amountMinor: number, currency: string): string {
  const amount = BigInt(amountMinor);
  const whole = amount / 100n;
  const fraction = (amount % 100n).toString().padStart(2, '0');
  return currency === 'EUR' ? `€${whole}.${fraction}` : `${whole}.${fraction} ${currency}`;
}
