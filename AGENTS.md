# WhatsNext Agent Instructions

## Project Overview

WhatsNext 是一个 AI 学习规划助手。

目标：
帮助用户管理课程学习、知识点、错题和长期学习路径。

核心模块：
- 用户系统
- 学习目标
- 知识库
- RAG检索
- 学习计划
- Agent决策


## Tech Stack

Backend:
- Go
- Gin
- MySQL
- Redis

Frontend:
- Vue3


## Architecture

Backend structure:

handler
    ↓
service
    ↓
repository
    ↓
database


Rules:
- Handler 不写业务逻辑
- Service 负责业务编排
- Repository 负责数据库访问


## Development Rules

Before implementing a feature:

1. Read PRD.md
2. Check ARCHITECTURE.md
3. Confirm existing data model


## Database

Database changes:

- Do not modify tables directly
- Update migration files


## AI Agent Rules

When implementing AI features:

- Separate RAG retrieval from Memory storage
- Do not put business state into prompt
- Store persistent user information separately


## Testing

Before finishing:

- Run backend tests
- Check API compatibility