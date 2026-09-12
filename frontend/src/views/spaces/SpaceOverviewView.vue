<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Calendar, ChatDotRound, Close, Delete, DocumentAdd, Download, FullScreen, Promotion, Refresh, Search, UploadFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type UploadRequestOptions } from 'element-plus'
import { learningSpacesApi } from '@/api/learning-spaces'
import { materialsApi } from '@/api/materials'
import { retrievalApi } from '@/api/retrieval'
import { learningAssetsApi } from '@/api/learning-assets'
import { chatApi } from '@/api/chat'
import { examApi } from '@/api/exam'
import { getApiErrorMessage } from '@/api/client'
import type { SpaceDetail } from '@/types/learning-space'
import type { MaterialIndexStatus, MaterialListItem } from '@/types/material'
import type { RetrievalResult } from '@/types/retrieval'
import type { KnowledgeArticle, KnowledgeNodeInput, LearningAssets, LearningNode } from '@/types/learning-asset'
import type { ChatMessage } from '@/types/chat'
import type { ExamAnalysis } from '@/types/exam'
import KnowledgeGraph from '@/components/KnowledgeGraph.vue'
import { getDaysUntil } from '@/utils/date'
import { renderMarkdown } from '@/utils/markdown'
import { resolveKnowledgeArticleBody } from '@/utils/handbook'

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const deleting = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const detail = ref<SpaceDetail>()
const materials = ref<MaterialListItem[]>([])
const indexStatuses = ref<Record<string, MaterialIndexStatus>>({})
const retrievalQuery = ref('')
const retrievalResults = ref<RetrievalResult[]>([])
const searching = ref(false)
const materialActionID = ref('')
const assets = ref<LearningAssets>({ knowledge_job: null, plan_job: null, knowledge_map: { nodes: [], edges: [] }, handbook: { articles: [] }, today_plan: { plan: null, stages: [], tasks: [] } })
const generatingAssets = ref(false)
const activeArticleId = ref('')
type WorkflowSection = 'materials' | 'map' | 'handbook' | 'patterns' | 'questions'

const activeSection = ref<WorkflowSection>('materials')
const workflowSections: { key: WorkflowSection; label: string; hint: string }[] = [
  { key: 'materials', label: '学习资料', hint: '导入与检索' },
  { key: 'map', label: '知识地图', hint: '构建知识结构' },
  { key: 'handbook', label: '知识手册', hint: '系统化学习' },
  { key: 'patterns', label: '题型手册', hint: '理解考察方式' },
  { key: 'questions', label: '真题题库', hint: '练习与反馈' },
]
const chatOpen = ref(false)
const chatInput = ref('')
const chatSending = ref(false)
const chatMessages = ref<ChatMessage[]>([])
const focusMode = ref(false)
const focusRunning = ref(false)
const focusElapsedSeconds = ref(0)
const focusTaskIndex = ref(0)
const contentFocusSentinel = ref<HTMLElement>()
let focusTimer: number | undefined
let contentFocusObserver: IntersectionObserver | undefined
const mapEditing = ref(false)
const nodeDialogOpen = ref(false)
const edgeDialogOpen = ref(false)
const editingNodeId = ref('')
const nodeForm = ref<KnowledgeNodeInput>({ name:'', node_type:'knowledge', description:'', exam_weight:0.5, estimated_minutes:30, position_x:null, position_y:null })
const edgeForm = ref({ from_node_id:'', to_node_id:'', relation_type:'prerequisite' })
const articleDialogOpen = ref(false)
const editingArticleId = ref('')
const articleForm = ref({ title:'', body:'' })
const examAnalysis = ref<ExamAnalysis>({patterns:[],questions:[],paper_count:0,answered_count:0,wrong_count:0})
let assetPollTimer: number | undefined
let pollTimer: number | undefined
const acceptedMaterialTypes = '.pdf,.pptx,.docx,.txt,.md,text/plain,text/markdown'
const supportedExtensions = new Set(['pdf', 'pptx', 'docx', 'txt', 'md'])
const daysLeft = computed(() => (detail.value ? getDaysUntil(detail.value.space.exam_date) : 0))
const indexedMaterialIDs = computed(() =>
  Object.values(indexStatuses.value)
    .filter((item) => item.status === 'indexed' || item.status === 'partial')
    .map((item) => item.material_id),
)
const studyMaterials = computed(()=>materials.value.filter(item=>item.material.material_kind!=='past_exam'))
const examMaterials = computed(() => materials.value.filter((item) => item.material.material_kind === 'past_exam'))
const activeExamMaterials = computed(() => examMaterials.value.filter(({ job }) => ['queued', 'processing'].includes(job.status)))

function examJobLabel(item: MaterialListItem) {
  if (item.job.status === 'queued') return '等待处理'
  if (item.job.status === 'processing') return 'AI 正在切分题目与归纳题型'
  if (item.job.status === 'failed') return item.job.error_message || item.material.failure_reason || '真题处理失败'
  return `已完成·${item.chunk_count} 个文本块`
}
const overallProgress = computed(() => {
  const nodes = assets.value.knowledge_map.nodes
  if (!nodes.length) return 0
  const totalWeight = nodes.reduce((sum, node) => sum + Math.max(node.exam_weight, 0.1), 0)
  const weightedScore = nodes.reduce((sum, node) => sum + node.mastery_score * Math.max(node.exam_weight, 0.1), 0)
  return Math.round(weightedScore / totalWeight)
})
const masteredNodeCount = computed(() => assets.value.knowledge_map.nodes.filter(node => node.mastery_status === 'mastered').length)
const remainingStudyMinutes = computed(() => Math.round(assets.value.knowledge_map.nodes.reduce(
  (sum, node) => sum + node.estimated_minutes * (1 - Math.min(node.mastery_score, 100) / 100),
  0,
)))
const estimatedStudyDays = computed(() => Math.max(1, Math.ceil(remainingStudyMinutes.value / Math.max(detail.value?.space.daily_minutes ?? 1, 1))))
const focusTask = computed(() => assets.value.today_plan.tasks[focusTaskIndex.value])
const focusTime = computed(() => {
  const hours = Math.floor(focusElapsedSeconds.value / 3600)
  const minutes = Math.floor((focusElapsedSeconds.value % 3600) / 60)
  const seconds = focusElapsedSeconds.value % 60
  return [hours, minutes, seconds].map(value => String(value).padStart(2, '0')).join(':')
})

function openFocusMode() {
  focusMode.value = true
  focusRunning.value = true
  window.clearInterval(focusTimer)
  focusTimer = window.setInterval(() => { if (focusRunning.value) focusElapsedSeconds.value += 1 }, 1000)
}

function closeFocusMode() {
  focusMode.value = false
  focusRunning.value = false
  window.clearInterval(focusTimer)
}

function nextFocusTask() {
  if (focusTaskIndex.value < assets.value.today_plan.tasks.length - 1) focusTaskIndex.value += 1
  else closeFocusMode()
}

function isWorkflowComplete(section: WorkflowSection) {
  if (section === 'materials') return indexedMaterialIDs.value.length > 0
  if (section === 'map') return assets.value.knowledge_map.nodes.length > 0
  if (section === 'handbook') return assets.value.handbook.articles.length > 0
  if (section === 'patterns') return examAnalysis.value.patterns.length > 0
  if (section === 'questions') return examAnalysis.value.answered_count > 0
  return false
}
function handleFocusKeydown(event: KeyboardEvent) { if (event.key === 'Escape' && focusMode.value) closeFocusMode() }
watch(contentFocusSentinel, (element) => {
  contentFocusObserver?.disconnect()
  document.body.classList.remove('workspace-focus')
  if (!element) return
  contentFocusObserver = new IntersectionObserver(([entry]) => {
    if (!entry) return
    document.body.classList.toggle('workspace-focus', !entry.isIntersecting && entry.boundingClientRect.top <= 1)
  }, { threshold: 0, rootMargin: '-1px 0px 0px' })
  contentFocusObserver.observe(element)
}, { flush: 'post' })

onMounted(async () => {
  window.addEventListener('keydown', handleFocusKeydown)
  try {
    const spaceId = String(route.params.spaceId)
    const [spaceDetail, materialList, learningAssets, exam] = await Promise.all([
      learningSpacesApi.get(spaceId),
      materialsApi.list(spaceId),
      learningAssetsApi.get(spaceId),
      examApi.get(spaceId),
    ])
    detail.value = spaceDetail
    materials.value = materialList.items
    assets.value = learningAssets
    examAnalysis.value = exam
    await refreshIndexStatuses()
    startPolling()
    if ([assets.value.knowledge_job, assets.value.plan_job].some((job) => job && ['queued', 'processing'].includes(job.status))) startAssetPolling()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally {
    loading.value = false
  }
})
onBeforeUnmount(() => { window.clearInterval(pollTimer); window.clearInterval(assetPollTimer); window.clearInterval(focusTimer); window.removeEventListener('keydown', handleFocusKeydown); contentFocusObserver?.disconnect(); document.body.classList.remove('workspace-focus') })

function startPolling() {
  window.clearInterval(pollTimer)
  if (!needsPolling()) return
  pollTimer = window.setInterval(() => void refreshProcessing(), 1500)
}

function needsPolling() {
  return (
    materials.value.some(({ job }) => job.status === 'queued' || job.status === 'processing') ||
    Object.values(indexStatuses.value).some((item) =>
      ['not_started', 'pending', 'indexing'].includes(item.status),
    )
  )
}

async function refreshProcessing() {
  await refreshJobs()
  await refreshIndexStatuses()
  if (!needsPolling()) window.clearInterval(pollTimer)
}

async function refreshJobs() {
  const spaceId = String(route.params.spaceId)
  let examCompleted = false
  const activeItems = materials.value.filter(
    ({ job }) => job.status === 'queued' || job.status === 'processing',
  )
  await Promise.all(
    activeItems.map(async (item) => {
      try {
        const job = await materialsApi.getJob(spaceId, item.job.id)
		examCompleted ||= item.material.material_kind === 'past_exam' && job.status === 'succeeded'
        item.job = job
        item.material.status =
          job.status === 'succeeded'
            ? 'ready'
            : job.status === 'failed'
              ? 'failed'
              : job.status
        item.material.failure_reason = job.error_message
      } catch {
        // Keep polling; transient API failures are surfaced by the next user action.
      }
    }),
  )
  if (examCompleted) {
    try {
      examAnalysis.value = await examApi.get(spaceId)
    } catch {
      // Keep polling material states; exam results can refresh on the next successful job or page load.
    }
  }
  if (!materials.value.some(({ job }) => job.status === 'queued' || job.status === 'processing')) {
    try {
      materials.value = (await materialsApi.list(spaceId)).items
      examAnalysis.value = await examApi.get(spaceId)
    } catch {
      // The terminal state is already visible; count refresh can wait for the next page load.
    }
  }
}

async function refreshIndexStatuses() {
  const spaceId = String(route.params.spaceId)
  const indexable = materials.value.filter((item) => item.material.status === 'ready')
  await Promise.all(
    indexable.map(async (item) => {
      try {
        indexStatuses.value[item.material.id] = await materialsApi.getIndexStatus(
          spaceId,
          item.material.id,
        )
      } catch {
        // A later polling round or page reload can recover transient dependency failures.
      }
    }),
  )
}

async function uploadMaterial(options: UploadRequestOptions, materialKind:'study'|'past_exam'='study') {
  const file = options.file
  const extension = file.name.split('.').pop()?.toLowerCase() || ''
  if (!supportedExtensions.has(extension)) {
    ElMessage.error('仅支持 PDF、PPTX、DOCX、TXT 和 Markdown')
    return
  }
  if (file.size > 100 * 1024 * 1024) {
    ElMessage.error('文件不能超过 100 MB')
    return
  }
  uploading.value = true
  uploadProgress.value = 0
  try {
    const result = await materialsApi.upload(String(route.params.spaceId), file, materialKind, (percentage) => {
      uploadProgress.value = percentage
    })
    materials.value.unshift(result)
    if (detail.value) detail.value.summary.material_count += 1
    options.onSuccess(result)
    ElMessage.success(materialKind==='past_exam'?'真题已上传，AI 正在切分题目与统计题型':'资料已上传，正在异步处理')
    startPolling()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally {
    uploading.value = false
  }
}
function uploadExamMaterial(options:UploadRequestOptions){return uploadMaterial(options,'past_exam')}

function materialTypeLabel(filename: string) {
  const extension = filename.split('.').pop()?.toUpperCase()
  return extension === 'MD' ? 'MD' : extension || 'FILE'
}

function formatBytes(bytes: number) {
  return bytes < 1024 * 1024
    ? `${Math.max(1, Math.round(bytes / 1024))} KB`
    : `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function statusLabel(item: MaterialListItem) {
  const labels = { queued: '等待处理', processing: '处理中', ready: '处理完成', failed: '处理失败' }
  return labels[item.material.status]
}

function indexLabel(materialId: string) {
  const status = indexStatuses.value[materialId]
  if (!status) return '等待索引状态'
  return {
    not_started: '等待索引',
    pending: '索引已排队',
    indexing: `索引中 ${status.progress}%`,
    indexed: '可检索',
    partial: `部分可检索（${status.indexed_chunks}/${status.total_chunks}）`,
    failed: '索引失败',
  }[status.status]
}

function isIndexActive(materialId: string) {
  const status = indexStatuses.value[materialId]
  return status ? ['pending', 'indexing'].includes(status.status) : false
}

function indexProgress(materialId: string) {
  return indexStatuses.value[materialId]?.progress ?? 0
}

async function focusHandbook(nodeId: string) {
  const article = assets.value.handbook.articles.find((item) => item.node_id === nodeId)
  if (!article) return
  activeSection.value = 'handbook'
  activeArticleId.value = article.id
  await nextTick()
  document.getElementById(`article-${article.id}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

async function retryMaterial(item: MaterialListItem) {
  materialActionID.value = item.material.id
  try {
    const result = await materialsApi.retry(String(route.params.spaceId), item.material.id)
    Object.assign(item, result)
    delete indexStatuses.value[item.material.id]
    ElMessage.success('已重新提交处理任务')
    startPolling()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally {
    materialActionID.value = ''
  }
}

async function downloadMaterial(item: MaterialListItem) {
  materialActionID.value = item.material.id
  try {
    const download = await materialsApi.getDownload(String(route.params.spaceId), item.material.id)
    window.location.assign(download.url)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally {
    materialActionID.value = ''
  }
}

async function removeMaterial(item: MaterialListItem) {
  try {
    await ElMessageBox.confirm(`确认删除“${item.material.original_name}”及其索引？`, '删除学习资料？', {
      confirmButtonText: '确认删除', cancelButtonText: '取消', type: 'warning',
    })
  } catch { return }
  materialActionID.value = item.material.id
  try {
    await materialsApi.remove(String(route.params.spaceId), item.material.id)
    materials.value = materials.value.filter((candidate) => candidate.material.id !== item.material.id)
    delete indexStatuses.value[item.material.id]
    if (detail.value) detail.value.summary.material_count -= 1
    if (item.material.material_kind === 'past_exam') {
      examAnalysis.value = await examApi.get(String(route.params.spaceId))
    }
    ElMessage.success(item.material.material_kind === 'past_exam' ? '真题已删除' : '资料已删除')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally { materialActionID.value = '' }
}

async function searchMaterials() {
  const query = retrievalQuery.value.trim()
  if (!query) return
  searching.value = true
  try {
    retrievalResults.value = (
      await retrievalApi.search(String(route.params.spaceId), query, indexedMaterialIDs.value)
    ).items
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally { searching.value = false }
}

async function generateLearningAssets() {
  generatingAssets.value = true
  try {
	assets.value.knowledge_job = await learningAssetsApi.generate(String(route.params.spaceId))
    ElMessage.success('已提交学习成果生成任务')
    startAssetPolling()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally { generatingAssets.value = false }
}

function startAssetPolling() {
  window.clearInterval(assetPollTimer)
  assetPollTimer = window.setInterval(() => void refreshAssetJobs(), 1500)
}

async function refreshAssetJobs() {
	const activeJobs = [assets.value.knowledge_job, assets.value.plan_job].filter((job) => job && ['queued', 'processing'].includes(job.status))
	if (!activeJobs.length) { window.clearInterval(assetPollTimer); return }
  try {
		let completed = false
		for (const job of activeJobs) {
			const updated = await learningAssetsApi.getJob(String(route.params.spaceId), job!.id)
			if (updated.job_type === 'knowledge_assets') assets.value.knowledge_job = updated
			else assets.value.plan_job = updated
			if (updated.status === 'succeeded') completed = true
			if (updated.status === 'failed') ElMessage.error(updated.error_message || '生成任务失败')
		}
		if (completed) {
			assets.value = await learningAssetsApi.get(String(route.params.spaceId))
      if (detail.value) {
        detail.value.summary.node_count = assets.value.knowledge_map.nodes.length
		detail.value.summary.mastered_node_count = assets.value.knowledge_map.nodes.filter(node=>node.mastery_status==='mastered').length
        detail.value.summary.today_task_count = assets.value.today_plan.tasks.length
      }
			ElMessage.success('生成任务已完成')
		}
  } catch {
    // Keep the last durable job state and retry on the next interval.
  }
}

async function generateLearningPlan() {
	generatingAssets.value = true
	try {
		assets.value.plan_job = await learningAssetsApi.generatePlan(String(route.params.spaceId))
		ElMessage.success('已提交学习计划生成任务')
		startAssetPolling()
	} catch (error) { ElMessage.error(getApiErrorMessage(error)) }
	finally { generatingAssets.value = false }
}

function openNodeEditor(node?:LearningNode){editingNodeId.value=node?.id??'';nodeForm.value=node?{name:node.name,node_type:node.node_type,description:node.description,exam_weight:node.exam_weight,estimated_minutes:node.estimated_minutes,position_x:node.position_x,position_y:node.position_y}:{name:'',node_type:'knowledge',description:'',exam_weight:0.5,estimated_minutes:30,position_x:120,position_y:120};nodeDialogOpen.value=true}
function editMapNode(nodeId:string){openNodeEditor(assets.value.knowledge_map.nodes.find(node=>node.id===nodeId))}
async function saveMapNode(){try{const spaceId=String(route.params.spaceId);if(editingNodeId.value)await learningAssetsApi.updateNode(spaceId,editingNodeId.value,nodeForm.value);else await learningAssetsApi.createNode(spaceId,nodeForm.value);assets.value=await learningAssetsApi.get(spaceId);nodeDialogOpen.value=false;ElMessage.success('知识节点已保存')}catch(error){ElMessage.error(getApiErrorMessage(error))}}
async function deleteMapNode(){if(!editingNodeId.value)return;try{await ElMessageBox.confirm('删除节点会同时删除其关系，并使当前计划失效。','删除知识节点？',{type:'warning'});await learningAssetsApi.deleteNode(String(route.params.spaceId),editingNodeId.value);assets.value.knowledge_map.nodes=assets.value.knowledge_map.nodes.filter(node=>node.id!==editingNodeId.value);assets.value.knowledge_map.edges=assets.value.knowledge_map.edges.filter(edge=>edge.from_node_id!==editingNodeId.value&&edge.to_node_id!==editingNodeId.value);assets.value.today_plan={plan:null,stages:[],tasks:[]};nodeDialogOpen.value=false;ElMessage.success('节点已删除')}catch(error){if(error!=='cancel')ElMessage.error(getApiErrorMessage(error))}}
async function saveNodePosition(nodeId:string,x:number,y:number){const node=assets.value.knowledge_map.nodes.find(item=>item.id===nodeId);if(node){node.position_x=x;node.position_y=y;node.user_edited=true}try{await learningAssetsApi.updateNodePosition(String(route.params.spaceId),nodeId,{position_x:x,position_y:y})}catch(error){ElMessage.error(getApiErrorMessage(error))}}
async function createMapEdge(){try{const edge=await learningAssetsApi.createEdge(String(route.params.spaceId),edgeForm.value);assets.value.knowledge_map.edges.push(edge);edgeDialogOpen.value=false;edgeForm.value={from_node_id:'',to_node_id:'',relation_type:'prerequisite'};ElMessage.success('知识关系已添加')}catch(error){ElMessage.error(getApiErrorMessage(error))}}
async function deleteMapEdge(edgeId:string){try{await learningAssetsApi.deleteEdge(String(route.params.spaceId),edgeId);assets.value.knowledge_map.edges=assets.value.knowledge_map.edges.filter(edge=>edge.id!==edgeId)}catch(error){ElMessage.error(getApiErrorMessage(error))}}
function handbookArticleBody(article:KnowledgeArticle){return resolveKnowledgeArticleBody(article,examAnalysis.value.patterns)}
function editHandbookArticle(article:KnowledgeArticle){editingArticleId.value=article.id;articleForm.value={title:article.title,body:handbookArticleBody(article)};articleDialogOpen.value=true}
async function saveHandbookArticle(){try{const updated=await learningAssetsApi.updateArticle(String(route.params.spaceId),editingArticleId.value,articleForm.value);const index=assets.value.handbook.articles.findIndex(article=>article.id===updated.id);if(index>=0)assets.value.handbook.articles[index]=updated;articleDialogOpen.value=false;ElMessage.success('手册章节已保存，后续 AI 生成会保留该内容')}catch(error){ElMessage.error(getApiErrorMessage(error))}}
async function markExamQuestion(questionId:string,isCorrect:boolean){try{const spaceId=String(route.params.spaceId);const result=await examApi.feedback(spaceId,questionId,isCorrect);const question=examAnalysis.value.questions.find(item=>item.id===questionId);if(question)question.is_correct=isCorrect;examAnalysis.value.answered_count=examAnalysis.value.questions.filter(item=>item.is_correct!==null).length;examAnalysis.value.wrong_count=examAnalysis.value.questions.filter(item=>item.is_correct===false).length;assets.value=await learningAssetsApi.get(spaceId);if(result.decision_status==='replanning'){startAssetPolling();ElMessage.success('反馈已更新掌握度，Agent 正在调整计划')}else{ElMessage.success('反馈已更新知识点掌握度')}}catch(error){ElMessage.error(getApiErrorMessage(error))}}
function masteryNode(nodeId:string){return assets.value.knowledge_map.nodes.find(node=>node.id===nodeId)}
function masteryLabel(status:string){return {unassessed:'未评估',weak:'薄弱',learning:'学习中',mastered:'已掌握'}[status]??'未评估'}
function masteryTagType(status:string){return status==='mastered'?'success':status==='weak'?'danger':status==='learning'?'warning':'info'}

async function sendChatMessage() {
	const message = chatInput.value.trim()
	if (!message || chatSending.value) return
	chatMessages.value.push({ id: crypto.randomUUID(), role: 'user', content: message })
	chatInput.value = ''
	chatSending.value = true
	try {
		const result = await chatApi.ask(String(route.params.spaceId), message)
		chatMessages.value.push({ id: crypto.randomUUID(), role: 'assistant', content: result.answer, sources: result.sources, decisionSignal: result.decision_signal })
		if (result.decision_signal.status === 'replanning') {
			ElMessage.success('该问答已纳入决策，正在自动调整学习计划')
			assets.value = await learningAssetsApi.get(String(route.params.spaceId))
			startAssetPolling()
		}
	} catch (error) {
		ElMessage.error(getApiErrorMessage(error))
	} finally { chatSending.value = false }
}

async function removeSpace() {
  if (!detail.value) return
  try {
    await ElMessageBox.confirm(
      `删除“${detail.value.space.name}”后，它将不再出现在学习空间列表中。`,
      '删除学习空间？',
      {
        confirmButtonText: '确认删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger',
      },
    )
  } catch {
    return
  }

  deleting.value = true
  try {
    await learningSpacesApi.remove(detail.value.space.id)
    ElMessage.success('学习空间已删除')
    await router.replace('/spaces')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error))
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="page-container">
    <el-skeleton v-if="loading" animated :rows="8" />
    <template v-else-if="detail">
      <header class="space-hero">
        <div>
          <div class="hero-tags">
            <span class="mode-badge">EXAM</span
            ><span
              ><el-icon><Calendar /></el-icon>{{ daysLeft }} 天后考试</span
            >
          </div>
          <h1 class="eva-title" :aria-label="`${detail.space.name}，袭来`">
            <span class="eva-title-main" aria-hidden="true">{{ detail.space.name }}</span>
            <span class="eva-title-punctuation" aria-hidden="true">、</span>
            <span class="eva-title-arrival" aria-hidden="true">袭来</span>
            <small aria-hidden="true">EPISODE: WHATSNEXT</small>
          </h1>
          <p>{{ detail.space.goal }}</p>
        </div>
        <div class="space-hero-actions">
          <el-button
            class="space-delete-action"
            type="danger"
            size="large"
            :icon="Delete"
            :loading="deleting"
            @click="removeSpace"
            >删除空间</el-button
          >
          <div class="space-upload-actions">
            <el-upload
              :accept="acceptedMaterialTypes"
              :show-file-list="false"
              :http-request="uploadMaterial"
              :disabled="uploading"
            >
              <el-button class="space-upload-action" size="large" :icon="UploadFilled" :loading="uploading"
                >{{ uploading ? `上传中 ${uploadProgress}%` : '学习资料' }}</el-button
              >
            </el-upload>
			    <el-upload :accept="acceptedMaterialTypes" :show-file-list="false" :http-request="uploadExamMaterial" :disabled="uploading"><el-button class="space-upload-action" size="large" :icon="UploadFilled" :loading="uploading">历年真题</el-button></el-upload>
          </div>
        </div>
      </header>
      <section class="tab-content-panel plan-result-panel featured-plan">
		<div class="section-heading"><div><p class="eyebrow">LEARNING PLAN</p><h2>{{ assets.today_plan.plan?.title || '个性化学习计划' }}</h2><p class="plan-dependency">AI 结合资料内容、知识结构、掌握度与真题反馈，持续调整整体路线和今日任务。</p></div><div class="plan-heading-action"><span>{{ assets.today_plan.plan?.total_minutes || 0 }} 分钟</span><el-button :icon="FullScreen" :disabled="!assets.today_plan.tasks.length" @click="openFocusMode">进入专注模式</el-button><el-button type="primary" :disabled="!assets.knowledge_map.nodes.length || !assets.handbook.articles.length" :loading="generatingAssets || !!(assets.plan_job && ['queued','processing'].includes(assets.plan_job.status))" @click="generateLearningPlan">{{ assets.today_plan.tasks.length ? '重新生成计划' : '生成学习计划' }}</el-button></div></div>
		<el-progress v-if="assets.plan_job && ['queued','processing'].includes(assets.plan_job.status)" :percentage="assets.plan_job.progress" />
        <div v-if="assets.knowledge_map.nodes.length" class="overall-plan">
          <section class="overall-progress-card">
            <div class="overall-progress-heading"><div><span>总体掌握进度</span><strong>{{ overallProgress }}%</strong></div><el-progress :percentage="overallProgress" :stroke-width="10" :show-text="false" /></div>
            <div class="overall-plan-stats">
              <div><strong>{{ masteredNodeCount }}/{{ assets.knowledge_map.nodes.length }}</strong><span>已掌握知识点</span></div>
              <div><strong>{{ remainingStudyMinutes }}</strong><span>预估剩余分钟</span></div>
              <div><strong>{{ estimatedStudyDays }}</strong><span>预估学习天数</span></div>
              <div><strong>{{ daysLeft }}</strong><span>距离考试天数</span></div>
            </div>
          </section>
          <section class="roadmap-card">
            <div class="plan-subheading"><div><span>AI 个性化总体路线</span><small>{{ assets.today_plan.plan?.generation_reason || '生成后将展示基于资料与学情的具体路线' }}</small></div></div>
            <div v-if="assets.today_plan.stages.length" class="roadmap-stages">
              <article v-for="(stage, index) in assets.today_plan.stages" :key="stage.id" :class="stage.status">
                <span class="roadmap-marker">{{ stage.status === 'complete' ? '✓' : index + 1 }}</span>
                <div><strong>{{ stage.title }}</strong><p>{{ stage.description }}</p></div>
                <small>{{ stage.status === 'complete' ? '已完成' : stage.status === 'active' ? '进行中' : '待开始' }} · {{ stage.estimated_days }} 天</small>
              </article>
            </div>
            <p v-else class="overview-plan-empty">点击“生成学习计划”，AI 将从你的资料知识点和当前掌握情况生成路线。</p>
          </section>
        </div>
        <div class="today-plan-heading"><div><span>TODAY</span><div><strong>今日执行清单</strong><small>个性化总体路线在今天的具体切片</small></div></div><strong>{{ assets.today_plan.plan?.total_minutes || 0 }} 分钟</strong></div>
        <div v-if="assets.today_plan.tasks.length" class="plan-task-list">
          <article v-for="(task, index) in assets.today_plan.tasks" :key="task.id"><el-checkbox /><div><strong>{{ task.title }}</strong><span>任务 {{ index + 1 }} · {{ task.estimated_minutes }} 分钟</span></div></article>
        </div>
        <el-empty v-else description="知识地图和手册就绪后，可在此生成学习计划。" />
      </section>
      <div ref="contentFocusSentinel" class="content-focus-sentinel" aria-hidden="true" />
      <nav class="learning-flow" aria-label="学习流程">
        <button
          v-for="(section, index) in workflowSections"
          :key="section.key"
          type="button"
          :class="{ active: activeSection === section.key, complete: isWorkflowComplete(section.key) }"
          :aria-current="activeSection === section.key ? 'step' : undefined"
          @click="activeSection = section.key"
        >
          <span class="flow-index">{{ isWorkflowComplete(section.key) ? '✓' : index + 1 }}</span>
          <span class="flow-copy"><strong>{{ section.label }}</strong><small>{{ section.hint }}</small></span>
        </button>
      </nav>
      <section v-if="activeSection === 'materials' && !studyMaterials.length" class="next-step-panel">
        <div class="step-icon">
          <el-icon><DocumentAdd /></el-icon>
        </div>
        <div>
          <p class="eyebrow">建议的下一步</p>
          <h2>上传第一份学习资料</h2>
          <p>支持 PDF、PPTX、DOCX、TXT 和 Markdown，WhatsNext 将为你构建知识地图。</p>
        </div>
        <el-upload :accept="acceptedMaterialTypes" :show-file-list="false" :http-request="uploadMaterial">
          <el-button type="primary">选择资料</el-button>
        </el-upload>
      </section>
      <section v-else-if="activeSection === 'materials'" class="materials-panel">
        <div class="section-heading">
          <div><p class="eyebrow">LEARNING MATERIALS</p><h2>学习资料</h2></div>
          <span>{{ studyMaterials.length }} 份</span>
        </div>
        <div class="material-list">
          <article v-for="item in studyMaterials" :key="item.material.id" class="material-item">
            <div class="material-icon">{{ materialTypeLabel(item.material.original_name) }}</div>
            <div class="material-copy">
              <strong>{{ item.material.original_name }}</strong>
              <span
                >{{ formatBytes(item.material.size_bytes) }} · {{ statusLabel(item) }}<template
                  v-if="item.chunk_count"
                > · {{ item.chunk_count }} 个文本块</template
                ></span
              >
              <el-progress
                v-if="item.job.status === 'queued' || item.job.status === 'processing'"
                :percentage="item.job.progress"
                :stroke-width="6"
              />
              <template v-if="item.material.status === 'ready'">
                <span class="index-state">{{ indexLabel(item.material.id) }}</span>
                <el-progress
                  v-if="isIndexActive(item.material.id)"
                  :percentage="indexProgress(item.material.id)"
                  :stroke-width="6"
                />
              </template>
              <small v-if="item.material.failure_reason" class="material-error">{{ item.material.failure_reason }}</small>
            </div>
            <div class="material-actions">
              <el-tag
                :type="item.material.status === 'ready' ? 'success' : item.material.status === 'failed' ? 'danger' : 'warning'"
                effect="light"
                >{{ statusLabel(item) }}</el-tag
              >
              <el-button-group>
                <el-button :icon="Download" title="下载" @click="downloadMaterial(item)" />
                <el-button
                  v-if="item.material.status === 'failed' || ['failed', 'partial'].includes(indexStatuses[item.material.id]?.status ?? '')"
                  :icon="Refresh"
                  title="重试"
                  :loading="materialActionID === item.material.id"
                  @click="retryMaterial(item)"
                />
                <el-button type="danger" :icon="Delete" title="删除" @click="removeMaterial(item)" />
              </el-button-group>
            </div>
          </article>
        </div>
      </section>

	  <section v-if="activeSection === 'patterns'" class="tab-content-panel exam-analysis-panel">
		<div class="section-heading"><div><p class="eyebrow">QUESTION PATTERN HANDBOOK</p><h2>题型手册</h2><p class="plan-dependency">AI 从历年真题中归纳的考察知识、解题路径、常见错误和考频。</p></div><div class="pattern-heading-actions"><el-button @click="activeSection='questions'">进入真题题库</el-button><el-upload :accept="acceptedMaterialTypes" :show-file-list="false" :http-request="uploadExamMaterial"><el-button type="warning" :icon="UploadFilled">继续上传真题</el-button></el-upload></div></div>
		<div v-if="examMaterials.length" class="exam-processing-list">
		  <article v-for="item in examMaterials" :key="item.material.id" class="exam-processing-item" :class="{ failed: item.job.status === 'failed' }">
			<div><strong>{{ item.material.original_name }}</strong><span>{{ examJobLabel(item) }}</span></div>
			<el-progress v-if="['queued','processing'].includes(item.job.status)" :percentage="item.job.progress" :stroke-width="7" />
			<el-tag v-else :type="item.job.status === 'succeeded' ? 'success' : 'danger'">{{ item.job.status === 'succeeded' ? '已完成' : '处理失败' }}</el-tag>
			<el-button-group><el-button :icon="Download" title="下载" @click="downloadMaterial(item)" /><el-button v-if="item.job.status === 'failed'" :icon="Refresh" title="重试" :loading="materialActionID === item.material.id" @click="retryMaterial(item)" /><el-button type="danger" :icon="Delete" title="删除真题" :loading="materialActionID === item.material.id" @click="removeMaterial(item)" /></el-button-group>
		  </article>
		</div>
		<div v-if="examAnalysis.patterns.length" class="exam-stats pattern-stats"><div><strong>{{ examAnalysis.paper_count }}</strong><span>份真题样本</span></div><div><strong>{{ examAnalysis.patterns.length }}</strong><span>个语义题型</span></div><div><strong>{{ examAnalysis.questions.length }}</strong><span>道题用于统计</span></div></div>
		<div v-if="examAnalysis.patterns.length" class="pattern-handbook handbook-layout"><nav class="handbook-toc"><strong>题型目录</strong><a v-for="(pattern,index) in examAnalysis.patterns" :key="pattern.id" :href="`#pattern-${pattern.id}`">{{ index+1 }}. {{ pattern.title }} <small>· {{ pattern.occurrence_count }} 次</small></a></nav><article class="handbook-content pattern-content"><section v-for="(pattern,index) in examAnalysis.patterns" :id="`pattern-${pattern.id}`" :key="pattern.id"><div class="handbook-chapter-meta"><span>第 {{ index+1 }} 类</span><el-tag :type="pattern.frequency_level==='high'?'danger':pattern.frequency_level==='medium'?'warning':'info'">{{ pattern.frequency_level==='high'?'高频':pattern.frequency_level==='medium'?'中频':'低频' }} · {{ pattern.occurrence_count }} 次</el-tag></div><h3>{{ pattern.title }}</h3><p class="pattern-summary">{{ pattern.description }}</p><div class="pattern-node-links"><strong>关联知识点</strong><div><button v-for="node in pattern.related_nodes" :key="node.node_id" type="button" :title="`${node.relation_reason}（置信度 ${node.confidence.toFixed(2)}）`" @click="focusHandbook(node.node_id)">{{ node.node_name }} <small>{{ Math.round(node.confidence*100) }}%</small></button></div></div><div class="pattern-section"><strong>考察知识</strong><p>{{ pattern.tested_knowledge }}</p></div><div class="pattern-section mistake"><strong>常见错误</strong><p>{{ pattern.common_mistakes }}</p></div><div class="pattern-section strategy"><strong>解题策略</strong><p>{{ pattern.solving_strategy }}</p></div></section></article></div>
		<el-empty v-else-if="!activeExamMaterials.length" description="还没有题型手册。上传一份真题后，AI 会自动归纳。" />
	  </section>

	  <section v-if="activeSection === 'questions'" class="tab-content-panel question-bank-panel">
		<div class="section-heading"><div><p class="eyebrow">PAST EXAM QUESTION BANK</p><h2>真题题库</h2><p class="plan-dependency">按题反馈你的真实掌握情况；答对和答错都会进入 Agent 决策。</p></div><div class="pattern-heading-actions"><el-button @click="activeSection='patterns'">查看题型手册</el-button><el-upload :accept="acceptedMaterialTypes" :show-file-list="false" :http-request="uploadExamMaterial"><el-button type="warning" :icon="UploadFilled">上传真题</el-button></el-upload></div></div>
		<div v-if="examMaterials.length" class="exam-processing-list">
		  <article v-for="item in examMaterials" :key="item.material.id" class="exam-processing-item" :class="{ failed: item.job.status === 'failed' }">
			<div><strong>{{ item.material.original_name }}</strong><span>{{ examJobLabel(item) }}</span></div>
			<el-progress v-if="['queued','processing'].includes(item.job.status)" :percentage="item.job.progress" :stroke-width="7" />
			<el-tag v-else :type="item.job.status === 'succeeded' ? 'success' : 'danger'">{{ item.job.status === 'succeeded' ? '已完成' : '处理失败' }}</el-tag>
			<el-button-group><el-button :icon="Download" title="下载" @click="downloadMaterial(item)" /><el-button v-if="item.job.status === 'failed'" :icon="Refresh" title="重试" :loading="materialActionID === item.material.id" @click="retryMaterial(item)" /><el-button type="danger" :icon="Delete" title="删除真题" :loading="materialActionID === item.material.id" @click="removeMaterial(item)" /></el-button-group>
		  </article>
		</div>
		<div v-if="examAnalysis.questions.length" class="exam-stats question-stats"><div><strong>{{ examAnalysis.questions.length }}</strong><span>题库总题数</span></div><div><strong>{{ examAnalysis.answered_count }}</strong><span>已反馈</span></div><div><strong>{{ examAnalysis.wrong_count }}</strong><span>当前错题</span></div></div>
		<div v-if="examAnalysis.questions.length" class="exam-question-list"><article v-for="question in examAnalysis.questions" :key="question.id"><div><span>第 {{ question.sequence_no }} 题 · {{ question.question_type }}</span><p>{{ question.stem }}</p></div><div class="question-feedback"><el-button :type="question.is_correct===true?'success':'default'" @click="markExamQuestion(question.id,true)">答对</el-button><el-button :type="question.is_correct===false?'danger':'default'" @click="markExamQuestion(question.id,false)">答错</el-button></div></article></div>
		<el-empty v-else-if="!activeExamMaterials.length" description="题库为空，请先上传历年真题。" />
	  </section>
      <section v-if="activeSection === 'materials'" class="retrieval-panel">
        <div class="section-heading">
          <div><p class="eyebrow">RAG RETRIEVAL</p><h2>资料检索</h2></div>
          <span>{{ indexedMaterialIDs.length }} 份资料可检索</span>
        </div>
        <el-input
          v-model="retrievalQuery"
          size="large"
          clearable
          placeholder="例如：TCP 为什么需要四次挥手？"
          :disabled="!indexedMaterialIDs.length"
          @keyup.enter="searchMaterials"
        >
          <template #append>
            <el-button :icon="Search" :loading="searching" @click="searchMaterials">检索</el-button>
          </template>
        </el-input>
        <p v-if="!indexedMaterialIDs.length" class="retrieval-hint">资料索引完成后即可使用。</p>
        <div v-if="retrievalResults.length" class="retrieval-results">
          <article v-for="result in retrievalResults" :key="result.chunk_id">
            <div>
              <strong>{{ result.material_name }}</strong>
              <span>相似度 {{ result.score.toFixed(3) }} · {{ result.source_type }} {{ result.source_start ?? '-' }}</span>
            </div>
            <p>{{ result.content }}</p>
          </article>
        </div>
        <el-empty v-else-if="retrievalQuery && !searching" :image-size="70" description="没有找到匹配内容" />
      </section>
      <section v-if="activeSection === 'map'" class="tab-content-panel">
		<div class="section-heading"><div><p class="eyebrow">KNOWLEDGE MAP</p><h2>知识地图</h2><p class="plan-dependency">基于已索引的学习资料构建，并同步生成知识手册。</p></div><div class="map-actions"><span>{{ assets.knowledge_map.nodes.length }} 个节点</span><el-button type="primary" :disabled="!indexedMaterialIDs.length" :loading="generatingAssets || !!(assets.knowledge_job && ['queued','processing'].includes(assets.knowledge_job.status))" @click="generateLearningAssets">{{ assets.knowledge_map.nodes.length?'重新生成':'生成知识成果' }}</el-button><el-button v-if="assets.knowledge_map.nodes.length" :type="mapEditing?'primary':'default'" @click="mapEditing=!mapEditing">{{ mapEditing?'完成编辑':'编辑地图' }}</el-button><el-button v-if="mapEditing" @click="openNodeEditor()">新增节点</el-button><el-button v-if="mapEditing" @click="edgeDialogOpen=true">新增关系</el-button></div></div>
		<el-progress v-if="assets.knowledge_job && ['queued','processing'].includes(assets.knowledge_job.status)" :percentage="assets.knowledge_job.progress" />
        <template v-if="assets.knowledge_map.nodes.length">
		  <p class="map-hint">{{ mapEditing?'编辑模式：点击节点修改内容，拖拽后位置会自动保存。':'拖拽、缩放或平移画布；点击节点定位到知识手册。' }}</p>
		  <KnowledgeGraph :nodes="assets.knowledge_map.nodes" :edges="assets.knowledge_map.edges" :editable="mapEditing" @node-click="focusHandbook" @node-edit="editMapNode" @node-position="saveNodePosition" />
		  <div v-if="mapEditing" class="map-edge-list"><article v-for="edge in assets.knowledge_map.edges" :key="edge.id"><span>{{ assets.knowledge_map.nodes.find(n=>n.id===edge.from_node_id)?.name }} → {{ assets.knowledge_map.nodes.find(n=>n.id===edge.to_node_id)?.name }} · {{ edge.relation_type }}</span><el-button text type="danger" @click="deleteMapEdge(edge.id)">删除</el-button></article></div>
        </template>
        <el-empty v-else :description="indexedMaterialIDs.length ? '点击“生成知识成果”构建知识地图。' : '请先完成学习资料索引。'" />
      </section>

	  <el-dialog v-model="nodeDialogOpen" :title="editingNodeId?'编辑知识节点':'新增知识节点'" width="min(32.5rem, calc(100vw - 2rem))"><el-form label-position="top"><el-form-item label="名称"><el-input v-model="nodeForm.name" /></el-form-item><el-form-item label="类型"><el-select v-model="nodeForm.node_type"><el-option label="知识" value="knowledge"/><el-option label="技能" value="skill"/><el-option label="练习" value="practice"/><el-option label="里程碑" value="milestone"/><el-option label="项目" value="project"/></el-select></el-form-item><el-form-item label="描述"><el-input v-model="nodeForm.description" type="textarea" :rows="4" /></el-form-item><div class="form-row"><el-form-item label="考试权重"><el-input-number v-model="nodeForm.exam_weight" :min="0" :max="1" :step="0.1" /></el-form-item><el-form-item label="预计分钟"><el-input-number v-model="nodeForm.estimated_minutes" :min="1" :max="1440" /></el-form-item></div></el-form><template #footer><el-button v-if="editingNodeId" type="danger" plain @click="deleteMapNode">删除节点</el-button><el-button @click="nodeDialogOpen=false">取消</el-button><el-button type="primary" @click="saveMapNode">保存</el-button></template></el-dialog>
	  <el-dialog v-model="edgeDialogOpen" title="新增知识关系" width="min(30rem, calc(100vw - 2rem))"><el-form label-position="top"><el-form-item label="起点"><el-select v-model="edgeForm.from_node_id" filterable><el-option v-for="node in assets.knowledge_map.nodes" :key="node.id" :label="node.name" :value="node.id"/></el-select></el-form-item><el-form-item label="关系"><el-select v-model="edgeForm.relation_type"><el-option label="前置" value="prerequisite"/><el-option label="相关" value="related"/></el-select></el-form-item><el-form-item label="终点"><el-select v-model="edgeForm.to_node_id" filterable><el-option v-for="node in assets.knowledge_map.nodes" :key="node.id" :label="node.name" :value="node.id"/></el-select></el-form-item></el-form><template #footer><el-button @click="edgeDialogOpen=false">取消</el-button><el-button type="primary" @click="createMapEdge">添加关系</el-button></template></el-dialog>

      <section v-if="activeSection === 'handbook'" class="handbook-document">
        <header><p class="eyebrow">KNOWLEDGE HANDBOOK</p><h2>{{ detail.space.name }}·知识手册</h2><p>基于已索引学习资料整理的完整知识总结。</p></header>
        <div v-if="assets.handbook.articles.length" class="handbook-layout">
          <nav class="handbook-toc"><strong>目录</strong><a v-for="(article, index) in assets.handbook.articles" :key="article.id" :href="`#article-${article.id}`">{{ index + 1 }}. {{ article.title }}</a></nav>
          <article class="handbook-content">
            <section v-for="(article, index) in assets.handbook.articles" :id="`article-${article.id}`" :key="article.id" :class="{ focused: activeArticleId === article.id }">
              <div class="handbook-chapter-meta"><span>第 {{ index + 1 }} 章 <em v-if="article.user_edited">已人工修改</em></span><el-button text type="primary" @click="editHandbookArticle(article)">编辑章节</el-button></div><h3>{{ article.title }}</h3><div v-if="masteryNode(article.node_id)" class="handbook-mastery"><div><el-tag :type="masteryTagType(masteryNode(article.node_id)!.mastery_status)">{{ masteryLabel(masteryNode(article.node_id)!.mastery_status) }}</el-tag><span>{{ masteryNode(article.node_id)!.evidence_count ? `基于 ${masteryNode(article.node_id)!.evidence_count} 道题的反馈` : '尚无作答证据' }}</span></div><el-progress :percentage="Math.round(masteryNode(article.node_id)!.mastery_score)" :status="masteryNode(article.node_id)!.mastery_status==='mastered'?'success':masteryNode(article.node_id)!.mastery_status==='weak'?'exception':undefined" /></div><div class="markdown-body" v-html="renderMarkdown(handbookArticleBody(article))"></div>
            </section>
          </article>
        </div>
        <el-empty v-else description="请先在知识地图步骤生成知识成果。" />
      </section>
	  <el-dialog v-model="articleDialogOpen" title="编辑手册章节" width="min(45rem, calc(100vw - 2rem))"><el-form label-position="top"><el-form-item label="章节标题"><el-input v-model="articleForm.title" /></el-form-item><el-form-item label="章节正文"><el-input v-model="articleForm.body" type="textarea" :rows="16" /></el-form-item></el-form><p class="edit-protection-hint">保存后该章节会标记为人工内容，重新生成时不会被 AI 覆盖。</p><template #footer><el-button @click="articleDialogOpen=false">取消</el-button><el-button type="primary" @click="saveHandbookArticle">保存章节</el-button></template></el-dialog>

		  <button class="chat-launcher" type="button" aria-label="打开 AI 学习助手" @click="chatOpen = true"><span class="chat-launcher-icon"><el-icon><ChatDotRound /></el-icon></span><span><strong>问问 WhatsNext AI</strong><small>基于当前资料解答</small></span><i>AI</i></button>
	  <el-drawer v-model="chatOpen" size="min(27.5rem, 100vw)" class="chat-drawer" :with-header="false">
		<div class="chat-shell">
		  <header><div class="chat-avatar"><el-icon><ChatDotRound /></el-icon></div><div><strong>WhatsNext AI</strong><span>基于当前学习空间回答</span></div></header>
		  <div class="chat-thread">
			<div v-if="!chatMessages.length" class="chat-welcome"><strong>有什么想问的？</strong><p>我会检索已索引的学习资料，并在回答中附上来源。</p><button type="button" @click="chatInput = '帮我总结这个学习空间的核心内容'">总结核心内容</button></div>
			<article v-for="message in chatMessages" :key="message.id" :class="['chat-message', message.role]"><div>{{ message.content }}</div><p v-if="message.decisionSignal" class="chat-decision-status">{{ message.decisionSignal.status === 'replanning' ? '已作为学习信号，Agent 正在重新规划' : '已作为学习信号，将在知识资产就绪后纳入计划' }}</p><details v-if="message.sources?.length"><summary>{{ message.sources.length }} 条资料来源</summary><a v-for="(source,index) in message.sources" :key="source.chunk_id" href="#" @click.prevent="activeSection='materials';chatOpen=false"><strong>[{{ index+1 }}] {{ source.material_name }}</strong><span>{{ source.source_type }} {{ source.source_start ?? '-' }}</span></a></details></article>
			<div v-if="chatSending" class="chat-typing"><i /><i /><i /></div>
		  </div>
		  <footer><el-input v-model="chatInput" type="textarea" :autosize="{minRows:2,maxRows:5}" resize="none" placeholder="输入你的问题…" @keydown.enter.exact.prevent="sendChatMessage" /><el-button type="primary" :icon="Promotion" :loading="chatSending" circle @click="sendChatMessage" /></footer>
		</div>
		  </el-drawer>
      <Teleport to="body">
        <Transition name="focus-fade">
          <div v-if="focusMode" class="focus-mode" role="dialog" aria-modal="true" aria-label="专注模式">
            <header><div class="brand"><span class="brand-mark">W</span><span>WhatsNext</span></div><button type="button" aria-label="退出专注模式" @click="closeFocusMode"><el-icon><Close /></el-icon><span>退出专注</span></button></header>
            <main>
              <div class="focus-context"><span>FOCUS SESSION</span><p>{{ detail.space.name }} · 任务 {{ focusTaskIndex + 1 }}/{{ assets.today_plan.tasks.length }}</p></div>
              <div class="focus-timer">{{ focusTime }}</div>
              <button class="focus-timer-toggle" type="button" @click="focusRunning = !focusRunning">{{ focusRunning ? '暂停计时' : '继续计时' }}</button>
              <section v-if="focusTask" class="focus-task-card">
                <span>当前任务</span>
                <h1>{{ focusTask.title }}</h1>
                <p>建议专注 {{ focusTask.estimated_minutes }} 分钟</p>
                <el-button type="primary" size="large" @click="nextFocusTask">{{ focusTaskIndex < assets.today_plan.tasks.length - 1 ? '完成并进入下一项' : '完成今日专注' }}</el-button>
              </section>
              <div class="focus-task-dots"><i v-for="(_, index) in assets.today_plan.tasks" :key="index" :class="{ done: index < focusTaskIndex, active: index === focusTaskIndex }" /></div>
            </main>
          </div>
        </Transition>
      </Teleport>
    </template>
  </div>
</template>
