# WhatsNext 系统架构设计

版本：v0.1  
状态：MVP 开发基线  
适用范围：Exam Mode

## 1. 架构目标

WhatsNext MVP 首先验证一条完整闭环：

```text
创建考试学习空间
  → 上传学习资料
  → 生成知识结构和知识手册
  → 生成当日学习计划
  → 提交学习反馈
  → 更新用户状态
  → 重新规划
```

架构优先级：

1. 业务状态可持久化、可追溯。
2. AI 生成结果必须有来源且可人工修正。
3. Planner 输出结构化计划，不只生成文本。
4. RAG 检索数据与用户学习状态分离。
5. MVP 保持单体部署，模块边界清晰，不过早拆分微服务。

## 2. 技术决策

| 领域 | 选型 | 说明 |
|---|---|---|
| 后端 | Go + Gin | REST API 与业务编排 |
| 数据访问 | `database/sql` + sqlx | 显式 SQL 与事务边界 |
| MySQL 驱动 | `go-sql-driver/mysql` | Go 连接 MySQL |
| Migration | golang-migrate | 版本化表结构变更 |
| 配置 | `caarlos0/env` | 环境变量解析与校验 |
| 日志 | `log/slog` | JSON 结构化日志 |
| 业务 ID | UUID v7 / `CHAR(36)` | 不可预测且按时间大致有序 |
| 前端 | Vue 3 + TypeScript + Vite | SPA |
| 组件库 | Element Plus | 表单、上传、树和任务界面 |
| 前端状态 | Pinia | 登录信息和跨页客户端状态 |
| 业务数据库 | MySQL | 用户、知识图、计划和反馈 |
| 缓存/队列 | Redis | 异步任务队列、限流和短期缓存 |
| 对象存储 | MinIO | 原始资料与生成文件，兼容 S3 API |
| 向量数据库 | Qdrant | 资料分块 Embedding 与相似度检索 |
| AI | 阿里云百炼（北京） | 通过 OpenAI-compatible Provider 接口隔离 |
| 认证 | JWT | 短期 Access Token + 可轮换 Refresh Token |
| 部署 | Docker Compose | API、Worker、Web、MySQL、Redis、MinIO |

默认模型为 `qwen3.7-plus`，低成本任务使用 `qwen3.6-flash`，Embedding 使用 `text-embedding-v4` 1024 维。模型 ID 不写死在业务代码中，由环境变量配置。

北京地域 OpenAI 兼容接口格式：

```text
https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1
```

## 3. 系统上下文

```text
浏览器
  │ HTTPS / JSON
  ▼
Vue SPA
  │ /api/v1
  ▼
Gin API
  ├─ MySQL       业务数据
  ├─ Redis       任务队列/缓存
  ├─ MinIO       原始文件
  ├─ Qdrant      向量检索
  └─ 百炼 API     提取、生成、Embedding

Worker
  ├─ Redis       消费任务
  ├─ MySQL       读写任务和知识资产
  ├─ MinIO       读取资料
  ├─ Qdrant      写入和查询向量
  └─ 百炼 API     AI 处理
```

API 与 Worker 复用同一 Go 代码库，以不同进程入口运行。

## 4. 代码结构

```text
backend/
├─ cmd/
│  ├─ api/
│  └─ worker/
├─ internal/
│  ├─ config/
│  ├─ handler/
│  ├─ service/
│  ├─ repository/
│  ├─ model/
│  ├─ dto/
│  ├─ middleware/
│  ├─ auth/
│  ├─ storage/
│  ├─ queue/
│  ├─ ai/
│  ├─ rag/
│  ├─ planner/
│  └─ job/
├─ migrations/
└─ tests/

frontend/
├─ src/
│  ├─ api/
│  ├─ assets/
│  ├─ components/
│  ├─ composables/
│  ├─ layouts/
│  ├─ router/
│  ├─ stores/
│  ├─ styles/
│  ├─ types/
│  ├─ utils/
│  └─ views/
└─ tests/
```

## 5. 后端分层规则

```text
Handler → Service → Repository → MySQL
                    ├─ Storage
                    ├─ Queue
                    └─ AI Provider
```

- Handler：解析请求、参数校验、调用 Service、转换 HTTP 响应。
- Service：权限校验、事务边界、业务规则和模块编排。
- Repository：只处理数据库访问，不包含 HTTP 或 AI 逻辑。
- Planner：使用结构化输入计算优先级和任务，不直接访问 HTTP。
- RAG：只负责分块、向量化和检索，不保存用户业务状态。

## 6. 核心领域模型

### 6.1 用户与认证

- `users`：用户账号、密码哈希和状态。
- `refresh_tokens`：只保存 Refresh Token 哈希、过期时间、撤销时间和设备信息。

### 6.2 学习空间

- `learning_spaces`：名称、模式、目标、考试日期、每日可用分钟数。
- MVP 只开放 `exam` 模式，字段保留未来 `growth` 的扩展能力。

### 6.3 资料与处理

- `learning_materials`：文件元数据、MinIO object key、处理状态。
- `generation_jobs`：异步任务类型、状态、进度、失败原因、重试次数。
- `material_chunks`：清洗后的文本分块、页码/幻灯片位置、内容哈希。
- `chunk_embeddings`：分块与 Qdrant point ID、Embedding 模型及索引状态的关联记录；向量本体存储在 Qdrant。

### 6.4 知识图

- `learning_nodes`：知识、技能、练习、里程碑或项目。
- `learning_edges`：`prerequisite`、`related` 或 `contains` 关系。
- `node_evidences`：节点与原始资料分块的来源关系。
- `knowledge_articles`：节点级知识手册，保存生成版本及用户编辑标记。

### 6.5 用户状态与计划

- `user_node_states`：掌握度、状态、错误次数、最近学习时间和置信度。
- `learning_plans`：一次规划的版本、日期、状态和生成原因。
- `plan_stages`：AI 结合资料知识点、掌握度、答题证据和剩余时间生成的个性化总体路线。
- `plan_nodes`：学习、复习、练习或测试任务。
- `learning_feedback`：完成情况、实际用时、难度、正确率和文本反馈。
- `state_change_logs`：用户状态变更前后值及变更原因。

所有业务表使用不可预测的全局 ID；所有隶属学习空间的查询必须同时校验 `user_id`。

## 7. 关键数据流

### 7.1 文件上传

1. 前端向 API 发送 multipart 文件。
2. API 检查类型、大小和空间权限。
3. API 流式写入 MinIO，不将整个文件常驻内存。
4. API 创建 Material 和 Job 记录，事务提交后投递 Redis 队列。
5. Worker 处理文件并持续更新 Job 进度。
6. 前端轮询 Job API，终态后停止。

### 7.2 AI 生成

```text
原始文件 → 文本提取 → 清洗/分块 → 节点候选
           → 去重/合并 → 关系生成 → 待用户确认
```

- AI 输出必须通过 JSON Schema 结构校验。
- 失败任务指数退避重试，超过上限后进入 `failed` 终态。
- 用户手动修改的内容不被后续 AI 任务静默覆盖。
- Prompt 不承载持久化业务状态；状态从数据库读取并作为结构化输入。

节点合并的 MVP 规则：

- 余弦相似度低于 `0.82`：不作为合并候选。
- `0.82` 至 `0.92`：由 `qwen3.7-plus` 判断并交由用户确认。
- 不低于 `0.92`：强候选，仍需模型置信度不低于 `0.90`、节点类型和上下文兼容，并由用户确认。
- MVP 不执行无人确认的自动合并。
- 阈值在积累真实确认数据后通过评测调整。

### 7.3 Planner

Planner 输入：

- Goal 和考试日期。
- Learning Graph。
- User Node State。
- 历史任务和反馈。
- 每日可用时间。

MVP 优先级为可解释的规则计算：

```text
priority = exam_weight
         × weakness
         × urgency
         × prerequisite_readiness
```

AI 可用于解释和辅助分析，不负责绕过时间与前置关系约束。

### 7.4 反馈与重新规划

1. 提交反馈并完成去重。
2. Service 更新 User Node State 并记录变更日志。
3. 已发布计划不原地改写，而是生成新版本。
4. 新版本保留调整原因，供前端展示。

MVP 在每次有效任务反馈提交后立即触发重新规划，队列会合并同一学习空间中尚未执行的重复任务。

## 8. 认证与安全

- 密码使用 Argon2id 哈希，不保存明文。
- Access Token 短期有效；Refresh Token 轮换，服务端只存哈希。
- Web 端 Refresh Token 放入 `HttpOnly`、`Secure`、`SameSite` Cookie。
- Access Token 由前端保存在内存中，不持久化到 localStorage。
- 文件下载经授权 API 或短期签名 URL，MinIO bucket 不公开。
- 上传检查 MIME、扩展名、大小和解析器安全性；MVP 单个文件最大 100 MB，PDF 最大 500 页。
- 百炼 API Key 只存在服务端环境变量中。
- 日志不记录 Token、密码、API Key 或完整资料文本。

## 9. 可靠性与一致性

- 数据库变更只能通过 migration。
- 异步任务至少传递一次，Worker 必须幂等。
- 关键写接口支持 `Idempotency-Key`。
- 文件上传成功但数据库写入失败时，记录清理任务回收孤立对象。
- Job 真实状态保存在 MySQL，Redis 只是传输和加速层。
- 百炼 API 失败不影响用户继续手动维护知识节点和计划。

## 10. 可观测性

- 所有请求使用 `X-Request-ID`。
- 结构化日志包含 request ID、user ID、space ID 和耗时，不包含敏感内容。
- 必备指标：HTTP 错误率、响应时间、队列长度、Job 成功率、AI 耗时、Token 消耗和估算成本。
- 健康检查：`/health/live` 只检查进程，`/health/ready` 检查 MySQL、Redis 和 MinIO。

## 11. 部署单元

Docker Compose 包含：

- `web`：Vue 构建产物与反向代理。
- `api`：Gin HTTP API。
- `worker`：Redis 任务消费者。
- `mysql`：业务数据库。
- `redis`：队列与缓存。
- `minio`：对象存储。
- `qdrant`：向量数据库。

生产环境只对外暴露 HTTPS 入口，MySQL、Redis、MinIO 和 Qdrant 不直接公网开放。

## 12. 测试策略

- Unit：Planner 计算、状态转换、鉴权和校验函数。
- Integration：Repository、migration、Redis 队列、MinIO 和 Qdrant。
- Contract：API 实现与 OpenAPI 文档一致。
- Frontend：Vitest + Vue Test Utils。
- E2E：Playwright 覆盖注册至重新规划的主流程。

## 13. 已确定与待确定事项

已确定：

- Gin、Vue 3、MySQL、Redis、MinIO、Qdrant、阿里云百炼（北京）、JWT、REST `/api/v1`、Docker Compose。
- 默认生成模型 `qwen3.7-plus`，低成本模型 `qwen3.6-flash`，Embedding 模型 `text-embedding-v4` 1024 维。
- 节点合并候选/强候选阈值分别为 `0.82`/`0.92`，模型置信度阈值为 `0.90`；MVP 不自动合并。
- 开发期 AI 月度预算上限 100 元，默认禁止免费额度用完后自动付费降级。
- MVP 单个文件最大 100 MB，PDF 最大 500 页。
- MVP 不设用户存储配额，但所有限制通过配置项提供。
- MVP 不实现邮箱验证、密码找回和第三方登录。
- 学习空间删除后保留 7 天恢复期，之后异步永久清理。
- 任务反馈后立即触发重新规划。

当前 MVP 架构必要的产品决策已确定。新的范围或成本决策出现时，实现前另行确认。
