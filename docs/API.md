# WhatsNext REST API 文档

版本：v0.1  
基础路径：`/api/v1`  
传输格式：HTTPS + JSON（文件上传除外）

## 1. 约定

### 1.1 认证

需登录的请求携带：

```http
Authorization: Bearer <access_token>
```

Refresh Token 通过 `HttpOnly` Cookie 传输，不在 JSON 中返回。

### 1.2 成功响应

```json
{
  "code": "OK",
  "message": "",
  "data": {}
}
```

- 创建成功：`201 Created`
- 普通成功：`200 OK`
- 已接受异步任务：`202 Accepted`
- 删除成功：`204 No Content`，无响应体

### 1.3 错误响应

```json
{
  "code": "VALIDATION_ERROR",
  "message": "request validation failed",
  "details": [
    {
      "field": "exam_date",
      "reason": "must be in the future"
    }
  ],
  "request_id": "req_01J..."
}
```

| HTTP | code | 说明 |
|---|---|---|
| 400 | `INVALID_ARGUMENT` | 参数或业务请求无效 |
| 400 | `VALIDATION_ERROR` | 字段校验失败 |
| 401 | `UNAUTHENTICATED` | 未登录或 Token 失效 |
| 403 | `PERMISSION_DENIED` | 无权操作资源 |
| 404 | `NOT_FOUND` | 资源不存在 |
| 409 | `CONFLICT` | 状态冲突或重复创建 |
| 413 | `FILE_TOO_LARGE` | 文件超过限制 |
| 415 | `UNSUPPORTED_FILE_TYPE` | 文件类型不支持 |
| 422 | `UNPROCESSABLE_CONTENT` | 语义正确但无法处理 |
| 429 | `RATE_LIMITED` | 请求过于频繁 |
| 500 | `INTERNAL_ERROR` | 未知服务端错误 |
| 502 | `AI_PROVIDER_ERROR` | AI 供应商失败 |
| 503 | `AI_QUOTA_EXCEEDED` | 百炼额度不足且未允许付费调用 |
| 503 | `SERVICE_UNAVAILABLE` | 依赖不可用 |

为避免泄露资源存在性，访问其他用户的学习空间及子资源统一返回 `404 NOT_FOUND`。

### 1.4 ID、时间与枚举

- ID 为不可预测字符串，客户端不解析其结构。
- 时间使用 RFC 3339 UTC，例如 `2026-08-03T08:30:00Z`。
- 纯日期使用 `YYYY-MM-DD`。
- 枚举在 JSON 中使用小写 snake_case 字符串。
- 金额如后续出现，使用最小货币单位整数。

### 1.5 分页

列表接口默认使用游标分页：

```http
GET /api/v1/spaces?limit=20&cursor=eyJpZCI6Ii4uLiJ9
```

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "items": [],
    "next_cursor": null
  }
}
```

- `limit` 默认 20，最大 100。
- `next_cursor` 为 `null` 表示没有下一页。

### 1.6 幂等

创建反馈、发布计划或触发生成任务时，客户端应携带：

```http
Idempotency-Key: <client-generated-unique-value>
```

相同用户、路由和 Key 在有效期内返回首次请求结果。

## 2. 公共与健康检查

### `GET /health/live`

进程存活检查，无需认证。

### `GET /health/ready`

服务就绪检查，检查 MySQL、Redis 和 MinIO，无需认证。

## 3. 认证 API

### `POST /auth/register`

请求：

```json
{
  "email": "user@example.com",
  "password": "correct horse battery staple",
  "display_name": "Kathy"
}
```

响应 `201`：

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "user": {
      "id": "usr_01J...",
      "email": "user@example.com",
      "display_name": "Kathy",
      "created_at": "2026-08-03T08:30:00Z"
    },
    "access_token": "<jwt>",
    "expires_in": 900
  }
}
```

Refresh Token 通过 Cookie 设置。

### `POST /auth/login`

```json
{
  "email": "user@example.com",
  "password": "correct horse battery staple"
}
```

响应与注册相同。

### `POST /auth/refresh`

使用 Refresh Token Cookie，轮换 Refresh Token 并返回新 Access Token。

### `POST /auth/logout`

撤销当前 Refresh Token 并清除 Cookie。返回 `204`。

### `GET /users/me`

返回当前用户资料。

### `PATCH /users/me`

```json
{
  "display_name": "New Name"
}
```

## 4. 学习空间 API

### 学习空间对象

```json
{
  "id": "spc_01J...",
  "name": "计算机网络期末复习",
  "mode": "exam",
  "goal": "期末考试达到 80 分",
  "exam_date": "2026-08-20",
  "daily_minutes": 120,
  "status": "active",
  "created_at": "2026-08-03T08:30:00Z",
  "updated_at": "2026-08-03T08:30:00Z"
}
```

### `POST /spaces`

```json
{
  "name": "计算机网络期末复习",
  "mode": "exam",
  "goal": "期末考试达到 80 分",
  "exam_date": "2026-08-20",
  "daily_minutes": 120
}
```

响应 `201`：学习空间对象。MVP 中 `mode` 只接受 `exam`。

### `GET /spaces`

可选参数：`status`、`limit`、`cursor`。

### `GET /spaces/{space_id}`

返回学习空间详情与聚合统计：

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "space": {},
    "summary": {
      "material_count": 3,
      "node_count": 42,
      "mastered_node_count": 8,
      "today_task_count": 5
    }
  }
}
```

### `PATCH /spaces/{space_id}`

允许更新 `name`、`goal`、`exam_date`、`daily_minutes`。`mode` 不可修改。

### `POST /spaces/{space_id}/archive`

将空间标记为 `archived`，不立即物理删除。

### `DELETE /spaces/{space_id}`

返回 `204`。空间进入 7 天软删除恢复期；恢复期结束后，Worker 异步删除 MySQL 业务数据、MinIO 对象和 Qdrant 向量。

### `POST /spaces/{space_id}/restore`

恢复 7 天保留期内的已删除空间。超过保留期返回 `404`。

## 5. 学习资料与任务 API

### `POST /spaces/{space_id}/materials`

`multipart/form-data`：

- `file`：文件。
- `kind`：`textbook | slides | notes | exam_paper`。

响应 `202`：

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "material": {
      "id": "mat_01J...",
      "file_name": "network.pdf",
      "kind": "textbook",
      "mime_type": "application/pdf",
      "size_bytes": 3456789,
      "status": "queued",
      "created_at": "2026-08-03T08:30:00Z"
    },
    "job": {
      "id": "job_01J...",
      "type": "process_material",
      "status": "queued",
      "progress": 0
    }
  }
}
```

MVP 支持 PDF、PPTX、DOCX、TXT 和 Markdown，单文件最大 100 MB，PDF 最大 500 页。超过大小限制返回 `413 FILE_TOO_LARGE`，页数超限在解析阶段返回 Job 失败原因 `PAGE_LIMIT_EXCEEDED`。MVP 不设用户总存储配额，限制仍保留为服务端可配置项。

### `GET /spaces/{space_id}/materials`

可选参数：`kind`、`status`、`limit`、`cursor`。

### `GET /spaces/{space_id}/materials/{material_id}`

返回资料元数据、处理状态和最近任务。

### `GET /spaces/{space_id}/materials/{material_id}/index-status`

返回当前配置 Embedding 模型的索引进度。状态来自 MySQL 中的持久化索引记录，不以 Redis 队列作为真实状态。

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "material_id": "mat_01J...",
    "status": "indexing",
    "embedding_model": "text-embedding-v4",
    "total_chunks": 20,
    "pending_chunks": 8,
    "indexed_chunks": 12,
    "failed_chunks": 0,
    "progress": 60,
    "job": {
      "id": "job_01J...",
      "job_type": "embed_material",
      "status": "processing",
      "progress": 58
    }
  }
}
```

`status` 取值：`not_started | pending | indexing | indexed | partial | failed`。`progress` 按已成功写入 Qdrant 的 Chunk 比例计算；`partial` 表示部分 Chunk 已索引、部分最终失败。

### `GET /spaces/{space_id}/materials/{material_id}/download-url`

返回短期 MinIO 签名地址：

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "url": "https://...",
    "expires_at": "2026-08-03T08:35:00Z"
  }
}
```

### `POST /spaces/{space_id}/materials/{material_id}/retry`

资料解析失败时重新创建 `process_material` Job；文本解析成功但索引失败时，仅重置失败 Chunk 并创建 `embed_material` Job。其他状态返回 `409 CONFLICT`。成功返回 `202` 和新 Job。

### `DELETE /spaces/{space_id}/materials/{material_id}`

若资料已被知识节点引用，默认返回 `409`，避免无意丢失来源。

### `GET /jobs/{job_id}`

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "id": "job_01J...",
    "type": "process_material",
    "status": "running",
    "stage": "extracting_nodes",
    "progress": 65,
    "attempt": 1,
    "error": null,
    "created_at": "2026-08-03T08:30:00Z",
    "started_at": "2026-08-03T08:30:03Z",
    "finished_at": null
  }
}
```

Job 状态：`queued | running | succeeded | failed | cancelled`。

### `POST /spaces/{space_id}/retrieval/search`

内部 RAG 检索接口，只返回带来源的 Chunk，不生成回答。

请求：

```json
{
  "query": "TCP 为什么需要四次挥手？",
  "top_k": 8,
  "material_ids": ["mat_01J..."]
}
```

- `query`：必填，最多 2000 个字符。
- `top_k`：可选，默认 8，范围 1–20。
- `material_ids`：可选，最多 20 个；所有资料必须属于当前用户和学习空间。

响应：

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "items": [
      {
        "chunk_id": "chk_01J...",
        "material_id": "mat_01J...",
        "material_name": "computer-network.pdf",
        "chunk_index": 12,
        "content": "...",
        "source_type": "page",
        "source_start": 35,
        "source_end": 36,
        "score": 0.87
      }
    ]
  }
}
```

检索强制使用 `user_id + learning_space_id` Qdrant Filter，并从 MySQL 回填 Chunk 正文和来源。未配置 Embedding 服务或 Qdrant 不可用时返回 `503 SERVICE_UNAVAILABLE`；Embedding Provider 调用失败返回 `502 AI_PROVIDER_ERROR`。

### `POST /spaces/{space_id}/chat`

在当前学习空间内提问。服务端先执行 RAG 检索，再将问题与检索结果交给对话生成器；对话不会把业务状态写入 Prompt，也不持久化为 Memory。

```json
{ "message": "帮我总结核心内容", "material_ids": [] }
```

返回 `answer`、用于生成该回答的 `sources` 和 `decision_signal`。每轮成功问答都会写入 `chat_learning_signals`，关联检索命中的知识节点，并自动触发重新规划。Planner 必须消费尚未纳入计划的信号，优先安排相关节点；新计划的 `generation_reason` 记录消费数量。当知识资产尚未就绪或已有任务执行时，信号保持 `pending_plan` 并在后续计划中消费。当前未配置真实模型时由 Mock 生成器返回可联调的带引用回答。

## 6. Learning Graph API

## 真题分析与反馈

资料上传接口的 multipart 表单新增 `material_kind`：

- `study`：教材、PPT、笔记等普通学习资料。
- `past_exam`：历年真题。

真题解析完成后，Worker 在同一事务中切分题目、按“考察知识 + 解题路径”归纳语义题型、累计跨试卷考频，并生成包含题型概述、考察知识、常见错误和解题策略的题型手册。“选择题/计算题/简答题”仅作为题目形式保存，不作为题型统计维度。题型同时生成为 `practice` 学习节点，因此不上传教材，只有一份真题也能建立计划。

- `GET /spaces/{space_id}/exam-analysis`：返回真题数、题型考频、切分后的题目及当前正误反馈。
- `PUT /spaces/{space_id}/exam-questions/{question_id}/feedback`：提交 `{ "is_correct": false, "note": "..." }`。

反馈按题目覆盖更新：错题作为权重 `+3` 的弱项信号，正确题作为权重 `-1` 的降权信号。两者都持续参与 Planner 决策，并在每次提交后自动触发重新规划。

题型与知识点通过 `exam_pattern_node_links` 多对多关联。每条关联保存 `confidence`、`relation_reason` 和 `source`；`GET exam-analysis` 在每个题型的 `related_nodes` 中返回关联知识点。一道错题的权重会同时作用于该题型关联的所有知识节点。

逐题反馈还会重新计算 `user_node_states`：保存掌握度、`unassessed | weak | learning | mastered` 状态、正错数、证据数和置信度。知识资产 API 在每个节点上返回该状态，知识手册以章节级标签和进度条展示。

### 学习成果聚合 API

### `POST /spaces/{space_id}/learning-assets/generate`

手动触发知识地图和知识手册生成任务。至少需要一个已完成索引的 Chunk；成功后会使旧学习计划失效。

### `POST /spaces/{space_id}/learning-plan/generate`

在知识地图和完整知识手册已发布后，独立触发学习计划生成。计划输入为结构化的知识节点、手册章节、学习目标和每日可用时间；前置成果不存在时返回 `409`。

### `GET /spaces/{space_id}/learning-assets/jobs/{job_id}`

返回生成任务的 `queued | processing | succeeded | failed` 状态与进度。

### `GET /spaces/{space_id}/learning-assets`

返回知识资产任务、计划任务及当前已发布内容：

```json
{
  "knowledge_job": { "id": "job_...", "job_type": "knowledge_assets", "status": "succeeded" },
  "plan_job": { "id": "job_...", "job_type": "learning_plan", "status": "succeeded" },
  "knowledge_map": { "nodes": [], "edges": [] },
  "handbook": { "articles": [] },
  "today_plan": { "plan": {}, "tasks": [] }
}
```

知识地图与手册在同一 MySQL 事务中替换；学习计划在后续独立事务中生成。

### 知识地图编辑 API

- `POST /spaces/{space_id}/knowledge-map/nodes`：新增节点，同时创建可继续编辑的手册章节。
- `PATCH /spaces/{space_id}/knowledge-map/nodes/{node_id}`：修改名称、类型、描述、权重和预计时间。
- `PATCH /spaces/{space_id}/knowledge-map/nodes/{node_id}/position`：持久化节点在画布中的坐标。
- `DELETE /spaces/{space_id}/knowledge-map/nodes/{node_id}`：删除节点及关系，并使当前计划失效。
- `POST /spaces/{space_id}/knowledge-map/edges`：新增 `prerequisite | related | contains` 关系。
- `DELETE /spaces/{space_id}/knowledge-map/edges/{edge_id}`：删除关系。

人工修改的节点和关系会记录 `user_edited=true`，重新生成时按节点合并并保留这些修改。

### 知识手册编辑 API

- `PATCH /spaces/{space_id}/knowledge-handbook/articles/{article_id}`：修改章节标题和完整正文。

保存后章节标记 `user_edited=true` 并自动触发重新规划。重新生成知识资产时采用逐节点合并：人工编辑的地图节点、关系和手册章节保留，仅替换未编辑的 AI 内容。

### Learning Node 对象

```json
{
  "id": "nod_01J...",
  "space_id": "spc_01J...",
  "name": "TCP 可靠传输",
  "type": "knowledge",
  "description": "TCP 可靠性机制",
  "exam_weight": 0.9,
  "estimated_minutes": 45,
  "source": "ai",
  "review_status": "pending",
  "user_edited": false,
  "created_at": "2026-08-03T08:30:00Z",
  "updated_at": "2026-08-03T08:30:00Z"
}
```

节点类型：`knowledge | skill | practice | milestone | project`。  
审核状态：`pending | accepted | rejected`。

### `POST /spaces/{space_id}/nodes`

```json
{
  "name": "TCP 可靠传输",
  "type": "knowledge",
  "description": "TCP 可靠性机制",
  "exam_weight": 0.9,
  "estimated_minutes": 45
}
```

手动创建的节点直接为 `accepted`。

### `GET /spaces/{space_id}/nodes`

可选参数：`type`、`review_status`、`query`、`limit`、`cursor`。

### `GET /spaces/{space_id}/nodes/{node_id}`

返回节点、用户状态、关联边和来源摘要。

### `PATCH /spaces/{space_id}/nodes/{node_id}`

允许更新 `name`、`type`、`description`、`exam_weight`、`estimated_minutes`。更新后 `user_edited=true`。

### `POST /spaces/{space_id}/nodes/{node_id}/review`

```json
{
  "decision": "accepted"
}
```

`decision` 为 `accepted | rejected`。

### `DELETE /spaces/{space_id}/nodes/{node_id}`

若节点已存在计划历史，使用软删除以保留历史解释。

### `POST /spaces/{space_id}/edges`

```json
{
  "from_node_id": "nod_parent",
  "to_node_id": "nod_child",
  "type": "prerequisite"
}
```

关系类型：`prerequisite | related | contains`。对有向关系进行环检测，违反约束返回 `409`。

### `DELETE /spaces/{space_id}/edges/{edge_id}`

返回 `204`。

### `GET /spaces/{space_id}/graph`

返回当前空间中已接受的节点和边：

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "nodes": [],
    "edges": [],
    "version": 7
  }
}
```

## 7. 来源与知识手册 API

### `GET /spaces/{space_id}/nodes/{node_id}/evidences`

返回资料名称、页码/位置、片段摘要和置信度。

### `GET /spaces/{space_id}/nodes/{node_id}/article`

返回节点级 Knowledge Book 文章：

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "id": "art_01J...",
    "node_id": "nod_01J...",
    "definition": "...",
    "core_mechanisms": ["..."],
    "common_mistakes": ["..."],
    "version": 2,
    "user_edited": false,
    "updated_at": "2026-08-03T08:30:00Z"
  }
}
```

### `PATCH /spaces/{space_id}/nodes/{node_id}/article`

人工编辑后设置 `user_edited=true`。

### `POST /spaces/{space_id}/nodes/{node_id}/article/generate`

创建或重新生成文章，返回 `202` 和 Job。当现有文章已被用户编辑时，必须传入：

```json
{
  "confirm_overwrite": true
}
```

## 8. 用户节点状态 API

### `GET /spaces/{space_id}/node-states`

可选参数：`status`、`node_id`、`limit`、`cursor`。

节点状态：

```json
{
  "node_id": "nod_01J...",
  "mastery": 0.4,
  "status": "need_review",
  "error_count": 5,
  "confidence": 0.7,
  "last_studied_at": "2026-08-02T08:30:00Z",
  "updated_at": "2026-08-03T08:30:00Z"
}
```

`mastery` 和 `confidence` 范围为 0～1。  
`status`：`not_started | learning | need_review | mastered`。

### `GET /spaces/{space_id}/nodes/{node_id}/state-history`

返回掌握度及状态变更历史。

MVP 不提供前端直接写入掌握度的接口；状态由反馈 Service 根据证据更新。

## 9. 学习计划 API

### `POST /spaces/{space_id}/plans/generate`

```json
{
  "date": "2026-08-04",
  "reason": "initial"
}
```

`reason`：`initial | user_requested | feedback_received | schedule_changed`。

对小型图可同步返回 `201`；若需 AI 辅助则返回 `202` 和 Job。客户端必须同时处理这两种成功响应。

### Learning Plan 对象

```json
{
  "id": "pln_01J...",
  "space_id": "spc_01J...",
  "date": "2026-08-04",
  "version": 3,
  "status": "active",
  "reason": "feedback_received",
  "change_summary": "因滑动窗口仍困难，增加 TCP 复习任务",
  "total_estimated_minutes": 115,
  "nodes": [
    {
      "id": "ptn_01J...",
      "learning_node_id": "nod_01J...",
      "action": "review",
      "title": "复习 TCP 滑动窗口",
      "priority": 0.92,
      "priority_reason": "高考试权重且掌握度低",
      "estimated_minutes": 35,
      "sequence": 1,
      "status": "pending"
    }
  ],
  "created_at": "2026-08-03T08:30:00Z"
}
```

计划状态：`draft | active | superseded | completed`。  
行动类型：`learn | review | practice | test`。  
任务状态：`pending | in_progress | completed | skipped`。

### `GET /spaces/{space_id}/plans/today`

返回当天活动计划；不存在时返回 `404`。

### `GET /spaces/{space_id}/plans`

可选参数：`date_from`、`date_to`、`status`、`limit`、`cursor`。

### `GET /spaces/{space_id}/plans/{plan_id}`

返回指定计划版本。

### `POST /spaces/{space_id}/plans/{plan_id}/activate`

将 draft 计划激活，并将同日上一个 active 版本标记为 superseded。

### `PATCH /spaces/{space_id}/plans/{plan_id}/nodes/{plan_node_id}`

```json
{
  "status": "in_progress"
}
```

任务完成建议使用反馈接口，以便同步更新用户状态。

## 10. 反馈与重新规划 API

### `POST /spaces/{space_id}/plans/{plan_id}/nodes/{plan_node_id}/feedback`

需要 `Idempotency-Key`。

```json
{
  "completion": "completed",
  "actual_minutes": 42,
  "difficulty": 4,
  "correct_rate": 0.6,
  "still_confused": true,
  "comment": "滑动窗口和累计确认仍然混淆"
}
```

- `completion`：`completed | partial | skipped`。
- `difficulty`：1～5，可空。
- `correct_rate`：0～1，可空。

响应 `201`：

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "feedback_id": "fbk_01J...",
    "state_change": {
      "node_id": "nod_01J...",
      "mastery_before": 0.5,
      "mastery_after": 0.3,
      "status_before": "learning",
      "status_after": "need_review",
      "reason": "high difficulty and still confused"
    },
    "replanning": {
      "triggered": true,
      "job_id": "job_01J..."
    }
  }
}
```

每次有效反馈均立即触发重新规划；队列中已有同一学习空间的待执行重规划任务时进行合并，避免频繁反馈产生重复计划。

### `GET /spaces/{space_id}/feedback`

可选参数：`node_id`、`date_from`、`date_to`、`limit`、`cursor`。

### `GET /spaces/{space_id}/plan-changes`

返回重新规划历史和每次调整原因。

## 11. MVP 前端使用的路由摘要

| 页面 | 主要 API |
|---|---|
| 注册/登录 | `/auth/register`, `/auth/login`, `/auth/refresh` |
| 学习空间列表 | `GET/POST /spaces` |
| 空间概览 | `GET /spaces/{id}`, `GET /plans/today` |
| 学习资料 | `/spaces/{id}/materials`, `/jobs/{id}` |
| 知识地图 | `/spaces/{id}/graph`, `/nodes`, `/edges` |
| 知识手册 | `/nodes/{id}/article`, `/evidences` |
| 今日计划 | `/plans/today`, `/feedback` |
| 计划历史 | `/plans`, `/plan-changes` |

## 12. API 实现顺序

1. Health + 统一响应/错误。
2. Auth + Users Me。
3. Learning Spaces。
4. Learning Nodes + Edges + Graph。
5. Plans + Feedback + User Node State。
6. Materials + Jobs + MinIO。
7. AI Generation + Evidences + Knowledge Articles。

前 5 步可先完成不依赖 AI 的手动规划闭环。

## 13. MVP 已确定产品限制

1. 单个文件最大 100 MB，PDF 最大 500 页。
2. MVP 不设用户总存储配额，但服务端保留配置能力。
3. MVP 不开启邮箱验证、密码找回和第三方登录。
4. 学习空间软删除后保留 7 天，之后异步永久清理。
5. 每次有效任务反馈立即触发重新规划。
6. Embedding 向量存储使用 Qdrant，业务关联信息保存在 MySQL。
7. AI 使用阿里云百炼北京地域：`qwen3.7-plus`、`qwen3.6-flash` 和 1024 维 `text-embedding-v4`。
8. 节点合并的候选/强候选阈值为 `0.82`/`0.92`，模型置信度阈值为 `0.90`；MVP 只建议合并。
9. 开发期 AI 月度预算上限 100 元，默认不在免费额度用完后自动转为付费调用。
