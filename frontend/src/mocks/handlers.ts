import { delay, http, HttpResponse } from 'msw'
import type { ApiEnvelope, CursorPage } from '@/types/api'
import type { AuthResult, User } from '@/types/auth'
import type { LearningSpace, SpaceDetail } from '@/types/learning-space'
import type { MaterialList } from '@/types/material'

const apiBase = '*/api/v1'
const sessionKey = 'whatsnext.mock.session'
const spacesKey = 'whatsnext.mock.spaces'

const demoUser: User = {
  id: 'usr_demo',
  email: 'demo@whatsnext.local',
  display_name: '学习者',
  created_at: new Date().toISOString(),
}

function ok<T>(data: T, status = 200) {
  return HttpResponse.json<ApiEnvelope<T>>({ code: 'OK', message: '', data }, { status })
}

function unauthorized() {
  return HttpResponse.json(
    { code: 'UNAUTHENTICATED', message: '请先登录', request_id: crypto.randomUUID() },
    { status: 401 },
  )
}

function readSpaces(): LearningSpace[] {
  const value = localStorage.getItem(spacesKey)
  if (value) return JSON.parse(value) as LearningSpace[]
  const sample: LearningSpace[] = [
    {
      id: 'spc_network',
      name: '计算机网络期末复习',
      mode: 'exam',
      goal: '期末考试达到 80 分',
      exam_date: new Date(Date.now() + 12 * 86400000).toISOString().slice(0, 10),
      daily_minutes: 120,
      status: 'active',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    },
  ]
  localStorage.setItem(spacesKey, JSON.stringify(sample))
  return sample
}

function authResult(user: User): AuthResult {
  sessionStorage.setItem(sessionKey, JSON.stringify(user))
  return { user, access_token: 'mock-access-token', expires_in: 900 }
}

export const handlers = [
  http.post(`${apiBase}/auth/register`, async ({ request }) => {
    await delay(300)
    const input = (await request.json()) as { email: string; display_name: string }
    return ok(
      authResult({ ...demoUser, email: input.email, display_name: input.display_name }),
      201,
    )
  }),
  http.post(`${apiBase}/auth/login`, async ({ request }) => {
    await delay(300)
    const input = (await request.json()) as { email: string }
    return ok(authResult({ ...demoUser, email: input.email }))
  }),
  http.post(`${apiBase}/auth/refresh`, () => {
    const value = sessionStorage.getItem(sessionKey)
    return value ? ok(authResult(JSON.parse(value) as User)) : unauthorized()
  }),
  http.post(`${apiBase}/auth/logout`, () => {
    sessionStorage.removeItem(sessionKey)
    return new HttpResponse(null, { status: 204 })
  }),
  http.get(`${apiBase}/spaces`, async () => {
    await delay(250)
    return ok<CursorPage<LearningSpace>>({ items: readSpaces(), next_cursor: null })
  }),
  http.post(`${apiBase}/spaces`, async ({ request }) => {
    await delay(350)
    const input = (await request.json()) as Omit<
      LearningSpace,
      'id' | 'status' | 'created_at' | 'updated_at'
    >
    const now = new Date().toISOString()
    const space: LearningSpace = {
      ...input,
      id: `spc_${crypto.randomUUID()}`,
      status: 'active',
      created_at: now,
      updated_at: now,
    }
    localStorage.setItem(spacesKey, JSON.stringify([space, ...readSpaces()]))
    return ok(space, 201)
  }),
  http.get(`${apiBase}/spaces/:spaceId`, async ({ params }) => {
    await delay(250)
    const space = readSpaces().find((item) => item.id === params.spaceId)
    if (!space) {
      return HttpResponse.json({ code: 'NOT_FOUND', message: '学习空间不存在' }, { status: 404 })
    }
    return ok<SpaceDetail>({
      space,
      summary: { material_count: 0, node_count: 0, mastered_node_count: 0, today_task_count: 0 },
    })
  }),
  http.get(`${apiBase}/spaces/:spaceId/materials`, () =>
    ok<MaterialList>({ items: [] }),
  ),
]
