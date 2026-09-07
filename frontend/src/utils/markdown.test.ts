import { describe, expect, it } from 'vitest'
import { normalizeMathMarkdown, renderMarkdown } from './markdown'

describe('math markdown rendering', () => {
  it('normalizes bare integral formulas and parenthesized exponents', () => {
    const source = 'y = e^(-∫P(x)dx) [C_1 + ∫Q(x)e^(∫P(x)dx)dx]'
    const normalized = normalizeMathMarkdown(source)

    expect(normalized).toContain('e^{-\\int P(x)\\,dx}')
    expect(normalized).toContain('C_{1}')
    expect(renderMarkdown(source)).toContain('katex-display')
  })
})
