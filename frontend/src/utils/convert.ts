export function convertToBytes(value: number, unit: string): number {
  const unitMultipliers: { [key: string]: number } = {
    'B': 1,
    'KB': 1024,
    'MB': 1024 * 1024,
    'GB': 1024 * 1024 * 1024,
    'TB': 1024 * 1024 * 1024 * 1024,
    'PB': 1024 * 1024 * 1024 * 1024 * 1024,
  };

  const multiplier = unitMultipliers[unit.toUpperCase()] || 1;
  return value * multiplier;
}

export function formatBytes(bytes: number, unit: string): string {
  if (bytes < 0) {
    return "-1";
  }
  if (unit === 'B' || unit === 'B/S' || unit === '%') {
    return bytes.toFixed(0);
  }
  return bytes.toFixed(2);
};