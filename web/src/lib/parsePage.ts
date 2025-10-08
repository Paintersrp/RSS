export function parsePage(value: unknown): number {
  const numeric = typeof value === 'string' ? Number.parseInt(value, 10) : Number(value)
  return Number.isFinite(numeric) && numeric > 0 ? numeric : 1
}
