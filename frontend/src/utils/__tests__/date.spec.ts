import { describe, expect, it } from 'vitest'
import { getDaysUntil } from '../date'

describe('getDaysUntil', () => {
  it('calculates calendar days from today', () => {
    expect(getDaysUntil('2026-08-13', new Date('2026-08-03T18:00:00'))).toBe(10)
  })

  it('does not return negative days', () => {
    expect(getDaysUntil('2026-08-01', new Date('2026-08-03T18:00:00'))).toBe(0)
  })
})
