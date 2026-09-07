import { describe, expect, it } from 'vitest'
import { compactGraphLayout } from './knowledge-graph-layout'
import type { LearningEdge, LearningNode } from '@/types/learning-asset'

describe('compact knowledge graph layout', () => {
  it('wraps a long prerequisite chain into a compact snake grid', () => {
    const nodes = Array.from({ length: 11 }, (_, index) => ({ id: `n${index}`, sort_order: index })) as LearningNode[]
    const edges = Array.from({ length: 10 }, (_, index) => ({
      id: `e${index}`,
      from_node_id: `n${index}`,
      to_node_id: `n${index + 1}`,
      relation_type: 'prerequisite',
    })) as LearningEdge[]
    const layout = compactGraphLayout(nodes, edges)

    expect(new Set([...layout.values()].map((item) => item.row)).size).toBe(3)
    expect(Math.max(...[...layout.values()].map((item) => item.x))).toBeLessThanOrEqual(830)
    expect(layout.get('n3')?.column).toBe(3)
    expect(layout.get('n4')?.column).toBe(3)
    expect(layout.get('n3')?.outgoingToNextRow).toBe(true)
    expect(layout.get('n4')?.incomingFromPreviousRow).toBe(true)
  })
})
