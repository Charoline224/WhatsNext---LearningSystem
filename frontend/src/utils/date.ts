const DAY_MS = 86_400_000

export function getDaysUntil(date: string, now = new Date()): number {
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const target = new Date(`${date}T00:00:00`).getTime()
  return Math.max(0, Math.ceil((target - today) / DAY_MS))
}
