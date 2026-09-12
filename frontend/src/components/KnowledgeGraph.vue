<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { MarkerType, Position, VueFlow, useVueFlow, type Edge, type Node, type NodeMouseEvent } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import type { LearningEdge, LearningNode } from '@/types/learning-asset'
import { compactGraphLayout } from '@/utils/knowledge-graph-layout'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'

const props = defineProps<{ nodes: LearningNode[]; edges: LearningEdge[]; editable?: boolean }>()
const emit = defineEmits<{ nodeClick: [nodeId: string]; nodeEdit: [nodeId:string]; nodePosition: [nodeId:string,x:number,y:number] }>()
const graphNodes = ref<Node[]>([])
const graphEdges = ref<Edge[]>([])
const { fitView } = useVueFlow()

const relationStyles = {
  prerequisite: { label: '前置', color: '#1a2744', dash: undefined },
  related: { label: '相关', color: '#4a5c3a', dash: '7 6' },
} as const

watch(
  () => [props.nodes, props.edges] as const,
  ([nodes, edges]) => {
	applyLayout(nodes, edges, true)
	void nextTick(() => fitView({ padding: 0.18, duration: 250 }))
  },
  { immediate: true, deep: true },
)

function applyLayout(nodes: LearningNode[], edges: LearningEdge[], preserveSaved: boolean) {
	const layout = compactGraphLayout(nodes, edges)
	const nextNodes: Node[] = nodes.map((node): Node => {
	  const item = layout.get(node.id)!
	  const useSaved = preserveSaved && node.position_x !== null && node.position_y !== null
      return {
        id: node.id,
        type: 'learning',
		position: { x: useSaved ? node.position_x! : item.x, y: useSaved ? node.position_y! : item.y },
		sourcePosition: item.outgoingToNextRow ? Position.Bottom : item.direction === 'forward' ? Position.Right : Position.Left,
		targetPosition: item.incomingFromPreviousRow ? Position.Top : item.direction === 'forward' ? Position.Left : Position.Right,
        data: node,
      }
    })
	const nextEdges: Edge[] = edges.map((edge): Edge => {
      const relation = relationStyles[edge.relation_type as keyof typeof relationStyles] ?? relationStyles.related
      return {
        id: edge.id,
        source: edge.from_node_id,
        target: edge.to_node_id,
        type: 'smoothstep',
        label: relation.label,
        markerEnd: { type: MarkerType.ArrowClosed, color: relation.color },
        style: { stroke: relation.color, strokeWidth: 2.5, strokeDasharray: relation.dash },
        labelStyle: { fill: relation.color, fontWeight: 600, fontSize: 11 },
        labelBgStyle: { fill: '#fff', fillOpacity: 0.9 },
	  }
	})
	graphNodes.value = nextNodes
	graphEdges.value = nextEdges
}

function resetLayout() {
	applyLayout(props.nodes, props.edges, false)
	void nextTick(() => fitView({ padding: 0.18, duration: 350 }))
}

function handleNodeClick(event: NodeMouseEvent) {
	if (props.editable) emit('nodeEdit', event.node.id)
	else emit('nodeClick', event.node.id)
}

function handleNodeDragStop(event: { node: Node }) { emit('nodePosition',event.node.id,event.node.position.x,event.node.position.y) }
</script>

<template>
  <div class="graph-shell">
    <VueFlow
      v-model:nodes="graphNodes"
      v-model:edges="graphEdges"
      :min-zoom="0.35"
      :max-zoom="1.8"
      fit-view-on-init
      :fit-view-on-init-options="{ padding: 0.22 }"
      @node-click="handleNodeClick"
	  @node-drag-stop="handleNodeDragStop"
    >
      <Background pattern-color="#dce1d7" :gap="22" />
      <Controls position="bottom-right" />
      <template #node-learning="{ data }">
        <button class="graph-node" type="button">
          <span class="graph-node-kind">{{ data.node_type === 'knowledge' ? '知识' : data.node_type }}</span>
          <strong>{{ data.name }}</strong>
          <small>{{ data.estimated_minutes }} 分钟 · 权重 {{ data.exam_weight }}</small>
		  <span class="graph-node-link">{{ editable ? '编辑节点 →' : '查看手册 →' }}</span>
        </button>
      </template>
    </VueFlow>
    <div class="graph-legend">
      <span><i class="prerequisite" />前置</span>
      <span><i class="related" />相关</span>
    </div>
	<button class="graph-layout-action" type="button" @click="resetLayout">自动排布</button>
  </div>
</template>

<style scoped>
.graph-shell { position: relative; height: 540px; overflow: hidden; border: 2px solid #4a5c3a; border-radius: 0; background: #fff; box-shadow: .25rem .25rem 0 rgba(251,191,36,.3); }
.vue-flow { width: 100%; height: 100%; }
.graph-node { display: grid; gap: 7px; width: clamp(10rem,18vw,12.8125rem); max-width: calc(100vw - 3rem); padding: 15px; border: 2px solid #4a5c3a; border-radius: 0; background: #fff; color: #172033; text-align: left; cursor: pointer; transition: transform .18s,box-shadow .18s; }
.graph-node:hover { transform: translate(.1rem,.1rem); box-shadow: .22rem .22rem 0 rgba(251,191,36,.4); }
.graph-node-kind { width: max-content; padding: 3px 7px; border: 1px solid #4a5c3a; border-radius: 0; background: #fbbf24; color: #172033; font-size: 10px; font-weight: 600; }
.graph-node strong { max-width: 100%; overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; line-height: 1.35; }
.graph-node small { color: #4f5968; }
.graph-node-link { color: #1a2744; font-size: 11px; font-weight: 600; }
.graph-legend { position: absolute; top: 12px; right: 12px; z-index: 5; display: flex; gap: 12px; padding: 8px 10px; border: 2px solid #4a5c3a; border-radius: 0; background: #fff; font-size: 11px; }
.graph-legend span { display: flex; align-items: center; gap: 5px; }
.graph-legend i { width: 1.125rem; border-top: 3px solid; }
.graph-legend .prerequisite { border-color: #1a2744; }
.graph-legend .related { border-color: #4a5c3a; border-top-style: dashed; }
.graph-layout-action { position: absolute; left: 12px; bottom: 12px; z-index: 6; padding: 8px 12px; border: 2px solid #4a5c3a; background: #fff; color: #172033; font-weight: 700; cursor: pointer; box-shadow: 2px 2px 0 rgba(251,191,36,.45); }
.graph-layout-action:hover { background: #fbbf24; }
:deep(.vue-flow__node) { border: 0; padding: 0; background: transparent; }
:deep(.vue-flow__node.selected .graph-node) { outline: 3px solid #fbbf24; }
</style>
