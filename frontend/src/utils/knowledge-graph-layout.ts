import type { LearningEdge, LearningNode } from '@/types/learning-asset'

export interface CompactNodeLayout {
  x: number
  y: number
  row: number
  column: number
  direction: 'forward' | 'backward'
  incomingFromPreviousRow: boolean
  outgoingToNextRow: boolean
}

export function topologicalNodeOrder(nodes: LearningNode[], edges: LearningEdge[]) {
  const nodeById = new Map(nodes.map((node) => [node.id, node]))
  const orderIndex = new Map(nodes.map((node, index) => [node.id, index]))
  const directed = edges.filter((edge) => edge.relation_type !== 'related' && nodeById.has(edge.from_node_id) && nodeById.has(edge.to_node_id))
  const incoming = new Map(nodes.map((node) => [node.id, 0]))
  const outgoing = new Map(nodes.map((node) => [node.id, [] as string[]]))
  for (const edge of directed) {
    incoming.set(edge.to_node_id, (incoming.get(edge.to_node_id) ?? 0) + 1)
    outgoing.get(edge.from_node_id)?.push(edge.to_node_id)
  }
  const queue = nodes.filter((node) => incoming.get(node.id) === 0).sort((a, b) => (a.sort_order ?? orderIndex.get(a.id)!) - (b.sort_order ?? orderIndex.get(b.id)!))
  const result: LearningNode[] = []
  while (queue.length) {
    const node = queue.shift()!
    result.push(node)
    for (const target of outgoing.get(node.id) ?? []) {
      const next = (incoming.get(target) ?? 0) - 1
      incoming.set(target, next)
      if (next === 0) {
        queue.push(nodeById.get(target)!)
        queue.sort((a, b) => (a.sort_order ?? orderIndex.get(a.id)!) - (b.sort_order ?? orderIndex.get(b.id)!))
      }
    }
  }
  if (result.length < nodes.length) {
    const placed = new Set(result.map((node) => node.id))
    result.push(...nodes.filter((node) => !placed.has(node.id)).sort((a, b) => a.sort_order - b.sort_order))
  }
  return result
}

export function compactGraphLayout(nodes: LearningNode[], edges: LearningEdge[], maxColumns = 4) {
  const ordered = topologicalNodeOrder(nodes, edges)
  const columns = Math.max(1, Math.min(maxColumns, Math.ceil(Math.sqrt(ordered.length * 1.5))))
  const layout = new Map<string, CompactNodeLayout>()
  ordered.forEach((node, index) => {
    const row = Math.floor(index / columns)
    const offset = index % columns
    const direction = row % 2 === 0 ? 'forward' : 'backward'
    const column = direction === 'forward' ? offset : columns - 1 - offset
    layout.set(node.id, {
      x: 65 + column * 255,
      y: 55 + row * 165,
      row,
      column,
      direction,
      incomingFromPreviousRow: false,
      outgoingToNextRow: false,
    })
  })
  for (const edge of edges.filter((item) => item.relation_type !== 'related')) {
    const source = layout.get(edge.from_node_id)
    const target = layout.get(edge.to_node_id)
    if (!source || !target || source.row === target.row) continue
    source.outgoingToNextRow = true
    target.incomingFromPreviousRow = true
  }
  return layout
}
