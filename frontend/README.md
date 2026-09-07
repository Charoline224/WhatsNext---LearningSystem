# WhatsNext Frontend

WhatsNext Exam Mode 的 Vue 3 SPA。当前已实现登录/注册、学习空间列表、创建空间和空间概览。

学习资料支持 PDF、PPTX、DOCX、TXT 和 Markdown，单文件最大 100 MB。

## 技术栈

- Vue 3 + TypeScript + Vite
- Vue Router + Pinia
- Axios + Element Plus + SCSS
- MSW（仅用于显式启用的独立测试）
- Vitest + Playwright

## 本地开发

```sh
corepack pnpm install
corepack pnpm dev
```

开发环境默认连接真实 Gin API。Vite 会把 `/api` 请求代理到 `http://127.0.0.1:8080`，因此先启动后端，再启动前端：

```sh
make backend-run
cd frontend && corepack pnpm dev
```

如后端使用其他地址，可设置：

```env
VITE_API_BASE_URL=/api/v1
```

若后端不在默认的 `127.0.0.1:8080`，可将 `VITE_API_BASE_URL` 设置为完整 API 地址，并同步配置后端 CORS。

只有在需要脱离后端运行 UI/E2E 测试时，才显式设置 `VITE_USE_MOCKS=true`。Mock 数据保存在浏览器 `localStorage`。

## 质量检查

```sh
corepack pnpm type-check
corepack pnpm lint
corepack pnpm test:unit -- --run
corepack pnpm build
```

首次运行 E2E 前安装 Chromium：

```sh
corepack pnpm exec playwright install chromium
corepack pnpm test:e2e -- --project=chromium
```
