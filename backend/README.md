# WhatsNext Backend

Go + Gin 单体后端，API 和 Worker 共用同一 Go Module，使用不同入口启动。

## 启动依赖

```sh
cp .env.example .env
make infra-up
make migrate-up
```

MySQL 在宿主机和 Compose 内部均固定使用 `3307` 端口；API、Worker 和 migration 容器通过 `mysql:3307` 访问 MySQL。

## 启动 API

```sh
cd backend
set -a
source .env
set +a
go run ./cmd/api
```

健康检查：

```text
GET http://localhost:8080/health/live
GET http://localhost:8080/health/ready
```

## MVP API

所有业务接口使用 `/api/v1` 前缀：

- `POST /auth/register`、`POST /auth/login`
- `POST /auth/refresh`、`POST /auth/logout`
- `GET /users/me`
- `GET /spaces`、`POST /spaces`
- `GET /spaces/:spaceId`、`PATCH /spaces/:spaceId`、`DELETE /spaces/:spaceId`
- `GET /spaces/:spaceId/materials`、`POST /spaces/:spaceId/materials`
- `GET /spaces/:spaceId/jobs/:jobId`
- `GET /spaces/:spaceId/materials/:materialId/chunks`
- `GET /spaces/:spaceId/materials/:materialId/index-status`
- `GET /spaces/:spaceId/materials/:materialId`、`GET /spaces/:spaceId/materials/:materialId/download-url`
- `POST /spaces/:spaceId/materials/:materialId/retry`、`DELETE /spaces/:spaceId/materials/:materialId`
- `POST /spaces/:spaceId/retrieval/search`
- `POST /spaces/:spaceId/chat`
- `GET /spaces/:spaceId/exam-analysis`
- `PUT /spaces/:spaceId/exam-questions/:questionId/feedback`
- `POST /spaces/:spaceId/learning-assets/generate`
- `POST /spaces/:spaceId/learning-plan/generate`
- `GET /spaces/:spaceId/learning-assets`、`GET /spaces/:spaceId/learning-assets/jobs/:jobId`
- `POST/PATCH/DELETE /spaces/:spaceId/knowledge-map/nodes...`
- `POST/DELETE /spaces/:spaceId/knowledge-map/edges...`

Access Token 通过 `Authorization: Bearer <token>` 发送；Refresh Token 仅保存在 HttpOnly Cookie 中并在每次刷新时轮换。创建学习空间可携带 `Idempotency-Key`，相同用户和 key 在 24 小时内返回同一资源。

认证相关配置：`JWT_SIGNING_KEY`（至少 32 字符）、`JWT_ISSUER`、`ACCESS_TOKEN_TTL` 和 `REFRESH_TOKEN_TTL`。

资料上传支持最大 100 MB 的 PDF、PPTX、DOCX、TXT 和 Markdown。服务端检查实际文件结构：Office Open XML 文件必须包含对应的主文档，文本必须是 UTF-8 且不能包含 NUL 字节。原文件流式写入 MinIO，任务写入 MySQL 后投递 Redis；Worker 即使错过队列消息，也会从 MySQL 恢复 queued 任务。

Worker 按 PDF 页、PPTX 幻灯片、DOCX 段落和文本行提取正文，统一清洗后生成约 1200 字符、长段落重叠 120 字符的 Chunk。Chunk 保存来源起止位置、字符数、token 粗估和 SHA-256；同一任务重试时以事务原子替换旧 Chunk。

扫描版 PDF 会在文本层不足时自动进入 OCR 兜底：Poppler 逐页渲染，Tesseract 使用中英文语料识别。OCR 采用可配置 Worker 池，默认并行处理 4 页，每页拥有独立超时和临时文件。

Embedding 基础设施使用百炼 OpenAI-compatible `/embeddings` 接口和 Qdrant REST API。向量只保存在 Qdrant，MySQL `chunk_embeddings` 表记录 Chunk、Point ID、模型、内容哈希和索引状态。相关配置见 `.env.example` 中的 `AI_EMBEDDING_*` 和 `QDRANT_*`。

未配置百炼时，开发环境可显式设置 `AI_MOCK_EMBEDDING=true`。Mock Provider 生成确定性特征哈希向量，仅用于联调 MySQL、Redis、MinIO 和 Qdrant 链路，不代表真实语义检索质量；生产环境禁止启用。

`AI_MOCK_GENERATION=true` 先使用已索引 Chunk 生成知识地图与手册，再使用已发布节点和手册章节独立生成学习计划。

Chunk 与 `embed_material` 任务在同一 MySQL 事务中创建。Worker 按批调用 Embedding，写入 Qdrant 后才将对应记录标记为 `indexed`；任务支持指数退避重试，Redis 通知丢失时仍会从 MySQL 恢复。

## 运行测试

```sh
go test ./...
```

启动基础设施后可运行真实依赖集成测试：

```sh
make integration-test
```

API 和 Worker 运行时，完整 HTTP 链路可通过 `make smoke-rag` 验证。

数据库结构只通过 `migrations/` 修改，不手动改表。
