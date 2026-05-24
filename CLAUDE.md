# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目定位

**LearnCode** — 一站式编程学习中心。用户指定一门编程语言，系统自动构建该语言的知识库（官方语法、标准库、编程规范），并提供隔离的运行时环境用于代码验证。核心机制是 LLM 根据用户学习进度**实时生成**小型练习项目（需求文档 + 参考答案 + 验证脚本），用户在自己的 IDE 中写完代码后提交目录，系统运行验证脚本判断通过与否。通过的项目自动纳入用户项目库，作为后续出题的学习背景参考。整个系统不存在预置题库，所有练习都是动态生成、用完即走的一次性产物。

## 当前实现状态（Phase 2 完成）

### 已完成的功能模块

**Language CRUD** — 语言大类管理完整可用：
- `GET /api/v1/languages` — 列表
- `GET /api/v1/languages/{id}` — 详情（含 icon、source_urls）
- `POST /api/v1/languages` — 创建
- `DELETE /api/v1/languages/{id}` — 删除（级联删除关联版本）
- `POST /api/v1/languages/init?step=query` — Wikipedia 验证 + LLM 分析
- `POST /api/v1/languages/init?step=confirm` — 确认并创建 Language（仅 Language，不含 Version）

**LLM 管理** — 多提供商配置 + Web UI：
- `GET/PUT /api/v1/config/llm` — 查看/修改 LLM 配置
- API 密钥自动遮罩（`sk-a...bcde`）；PUT 时智能保留（名称匹配 + 指纹匹配，支持重命名）
- 每个提供商支持多个模型（`models: []string`）
- 前端 `/settings` 页面管理全部提供商

**前端框架**：
- Tailwind CSS v4 暗色主题（`@import "tailwindcss"` + `@theme`，无 PostCSS）
- 页面：HomePage、LanguagesPage、LanguageDetailPage、AddLanguagePage、SettingsPage、NotFoundPage
- 字体：Inter（正文）+ JetBrains Mono（代码）

**语言初始化流程**（Wikipedia 验证）：
- 用户输入语言名 → Go 调用 Wikipedia opensearch API → 获取分类 + infobox 数据
- Go 层信号评分直接拒绝非语言（framework/library 分类 → 不消耗 LLM token）
- LLM 基于 Wikipedia 数据分类（language_analyze.yaml）→ 输出 is_language + confidence
- 可选：LLM 从官网页面提取 docs/runtime URL（language_resources.yaml）
- 仅返回确认卡片（icon 暂用占位），用户只能确认或取消，不可编辑
- 确认后只创建 Language（Version 是后续步骤）

### 数据库

**迁移文件**：
- `server/migrations/001_initial_schema.sql` — languages + language_versions 表
- `server/migrations/002_add_language_icon.sql` — languages 添加 icon, source_urls 列

**Language 表字段**：id, name, slug, icon, compatibility_model, source_urls, created_at

### LLM 抽象层

- Provider 接口：`Chat(ctx, ChatRequest) (*ChatResponse, error)` + `Name() string`
- 纯 `net/http`，不引入 SDK，调用 `/chat/completions` 端点
- 多 provider + fallback 链 + token 用量跟踪（sync.Mutex）
- Prompt 模板：YAML 文件 + Go `text/template`，存储在 `server/prompts/`

### 当前 Prompt 文件

| 文件 | 用途 |
|------|------|
| `language_analyze.yaml` | LLM 基于 Wikipedia 数据判定编程语言分类 |
| `language_resources.yaml` | LLM 从官网页面提取 docs/runtime URL |

## 技术栈选型

### 后端：Go + chi v5 + sqlx + PostgreSQL

- HTTP 路由：`github.com/go-chi/chi/v5`
- 数据库：`github.com/jmoiron/sqlx` + `github.com/lib/pq`
- 配置：`gopkg.in/yaml.v3`
- HTML 解析：`golang.org/x/net/html`
- API 响应格式：`{ok: bool, data?: T, error?: string}`

### 前端：React + TypeScript + Vite + Tailwind CSS v4

- 路由：react-router-dom v7
- Vite proxy：`/api` → `localhost:8080`
- Tailwind v4 无 PostCSS 配置，通过 `@tailwindcss/vite` 插件

## 项目架构（实际文件）

```
learncode/
├── server/
│   ├── cmd/learncode/main.go       # 入口
│   ├── internal/
│   │   ├── api/
│   │   │   ├── language.go         # LanguageHandler (CRUD + Init)
│   │   │   ├── version.go          # VersionHandler
│   │   │   ├── config.go           # ConfigHandler (LLM 配置管理)
│   │   │   ├── response.go         # RespondJSON / RespondError
│   │   │   └── middleware.go       # Recovery, Logging, CORS
│   │   ├── service/
│   │   │   ├── language_service.go # Language CRUD + CreateLanguageInput
│   │   │   ├── language_init_service.go  # Query(多步验证) + Confirm
│   │   │   ├── language_init_service_test.go
│   │   │   └── version_service.go  # Version CRUD
│   │   ├── model/
│   │   │   └── language.go         # Language + LanguageVersion structs
│   │   ├── repo/
│   │   │   ├── db.go               # NewDB + RunMigrations
│   │   │   ├── language_repo.go    # LanguageRepo (List/GetByID/GetBySlug/Create/Delete)
│   │   │   └── version_repo.go     # VersionRepo
│   │   ├── llm/
│   │   │   ├── types.go            # Provider interface, ChatRequest/Response
│   │   │   ├── provider.go         # openAIProvider (OpenAI 兼容实现)
│   │   │   ├── service.go          # Service (多 provider + fallback + token 统计)
│   │   │   ├── template.go         # LoadTemplate (YAML + text/template)
│   │   │   └── template_test.go
│   │   ├── scraper/
│   │   │   ├── scraper.go          # Client (HTTP + Wikipedia base URL)
│   │   │   ├── wikipedia.go        # SearchWikipedia, GetPageCategories, GetInfobox, ScoreSignal
│   │   │   └── fetcher.go          # FetchPageText (HTML → 纯文本)
│   │   └── config/
│   │       ├── config.go           # Load, Save, LLMConfig, MaskKey, ToResponse
│   │       └── config_test.go
│   ├── prompts/
│   │   ├── language_analyze.yaml   # LLM 基于 Wikipedia 数据判定编程语言
│   │   └── language_resources.yaml # LLM 从官网提取 docs/runtime URL
│   ├── migrations/
│   │   ├── 001_initial_schema.sql
│   │   └── 002_add_language_icon.sql
│   └── config.yaml
├── web/
│   ├── src/
│   │   ├── api/client.ts           # 类型化 API 客户端 (Language, InitSuggestion, LLMConfig, etc.)
│   │   ├── components/Layout.tsx   # 暗色 sticky header + 导航
│   │   ├── pages/
│   │   │   ├── HomePage.tsx
│   │   │   ├── LanguagesPage.tsx
│   │   │   ├── LanguageDetailPage.tsx  # 含删除按钮
│   │   │   ├── AddLanguagePage.tsx     # Wikipedia 验证 + 只读确认卡片
│   │   │   ├── SettingsPage.tsx        # LLM 配置管理（多模型支持）
│   │   │   └── NotFoundPage.tsx
│   │   └── App.tsx
│   └── vite.config.ts              # Tailwind v4 plugin + /api proxy
└── CLAUDE.md
```

## 核心数据模型（已实现部分）

```
Language
  ├─ id, name, slug, icon
  ├─ compatibility_model: "strict" | "versioned"
  ├─ source_urls: jsonb { docs, runtime }
  └─ created_at

LanguageVersion
  ├─ id, language_id → Language
  ├─ version, status: "active" | "archived"
  ├─ runtime_config: jsonb, source_urls: jsonb
  ├─ initialized: bool, last_version_check_at
  └─ created_at, updated_at
```

### 设计原则

- `compatibility_model`："strict" 语言一个活跃版本（Go, Java），"versioned" 语言多版本并存（Python, C++, Rust）
- UserProject 锚定到 LanguageVersion
- 用户学习画像以 Language 为单元跨版本计算

## 开发约定

- Go 三层架构：handler → service → repo，标准库风格，无重量框架
- 数据库：sqlx + 原生 SQL，不引入 ORM
- LLM：纯 `net/http` 调用 OpenAI 兼容 `/chat/completions`，不引入 SDK
- 提示词：YAML 模板 + Go `text/template`，温度 0.1，短小精悍
- 前端：React Context + useReducer，不引入 Redux
- API 响应：`{ok: bool, data?: T, error?: string}`
- Wikipedia User-Agent：`LearnCode/1.0`
- 执行安全：Docker 容器隔离（优先），`os/exec` + ulimit（回退），默认 30s 超时，无网络

## 关键命令

```bash
# 开发环境
docker-compose up -d                    # PostgreSQL
cd server && go run ./cmd/learncode    # 后端 :8080
cd web && npm run dev                  # 前端 :5173

# 测试
go test ./...                          # 全部测试
cd web && npx tsc -b --noEmit          # 前端类型检查

# 构建
cd server && go build -o learncode ./cmd/learncode
```

## 下一阶段任务

用户提到的下一步："完成编程语言初始化后，实现版本管理（容器环境构建）"。在版本管理之前，用户打算先重构前端并修复一些前端问题。
