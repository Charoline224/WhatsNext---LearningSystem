export type SpaceStatus = 'active' | 'archived'

export interface LearningSpace {
  id: string
  name: string
  mode: 'exam'
  goal: string
  exam_date: string
  daily_minutes: number
  status: SpaceStatus
  created_at: string
  updated_at: string
}

export interface CreateSpaceInput {
  name: string
  mode: 'exam'
  goal: string
  exam_date: string
  daily_minutes: number
}

export interface SpaceSummary {
  material_count: number
  node_count: number
  mastered_node_count: number
  today_task_count: number
}

export interface SpaceDetail {
  space: LearningSpace
  summary: SpaceSummary
}
