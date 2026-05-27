# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目定位

**LearnCode** — 一站式编程学习中心。用户指定一门编程语言，系统自动构建该语言的知识库（官方语法、标准库、编程规范，形成小型语言 wiki），并提供隔离的运行时环境用于代码验证。核心机制是 LLM 根据用户学习进度**实时生成**小型练习项目（需求文档 + 参考答案 + 验证脚本），参考答案与验证脚本对用户不可见。出题分两步验证：第一步 LLM 确认验证脚本对需求文档的覆盖充分且正确，第二步 LLM 确认参考答案可跑通验证脚本（否则修改参考答案）。用户在自己的 IDE 中按需求文档写完代码后提交目录，系统在隔离容器中运行验证脚本判断通过与否。通过的项目自动纳入用户项目库，作为后续出题的学习背景参考（避免重复、逐步提升难度）。整个系统不存在预置题库，所有练习都是动态生成、用完即走的一次性产物。

知识库具有双重用途：前端以层级结构展示给用户浏览，数据库侧供 LLM 检索以辅助出题和判题（后期接入 embedding/rerank 模型提升检索精度）。

## 当前实现状态

### 已完成的功能模块

**Language CRUD** — 语言大类管理完整可用：
- `GET /api/v1/languages` — 列表
- `GET /api/v1/languages/{id}` — 详情（含 icon、source_urls）
- `POST /api/v1/languages` — 创建
- `DELETE /api/v1/languages/{id}` — 删除（级联删除关联版本）
- `POST /api/v1/languages/init` — 三阶段初始化流水线（分析 → 调研 → 描述），每阶段可无限轮 LLM 调用

**LLM 管理** — 多提供商配置 + Web UI：
- `GET/PUT /api/v1/config/llm` — 查看/修改 LLM 配置
- API 密钥自动遮罩（`sk-a...bcde`）；PUT 时智能保留（名称匹配 + 指纹匹配，支持重命名）
- 每个提供商支持多个模型（`models: []string`）
- 前端 `/settings` 页面管理全部提供商

**代码执行** — 隔离运行时验证：
- `POST /api/v1/execute` — 提交代码在容器中执行
- Docker 容器隔离（优先），`os/exec` + ulimit（回退），默认 30s 超时，无网络
- 前端 Playground 嵌入式在线编辑和运行

**前端框架**：
- Tailwind CSS v4 暗色主题（`@import "tailwindcss"` + `@theme`，无 PostCSS）
- 页面：HomePage、LanguagesPage、LanguageDetailPage、AddLanguagePage、SettingsPage、NotFoundPage
- LanguageDetailPage 含嵌入式 Playground 和知识库内容展示
- 字体：Inter（正文）+ JetBrains Mono（代码）

**语言初始化流程**（三阶段流水线）：
- 阶段一（分析）：Wikipedia 搜索验证 → Go 层信号评分拒绝非语言 → LLM 分类判定 is_language + confidence，同时判定兼容模式（多版本并存 / 向后兼容 / 无版本区分）
- 阶段二（调研）：LLM 查找最新稳定版本（版本号 + 环境来源：dockerhub / 官网下载页 / 包管理器），全部需确认有效；递归获取历史版本
- 阶段三（描述）：LLM 生成语言描述、icon 描述、生态信息等结构化数据
- 每阶段支持多轮 LLM 调用，直到结果满足要求

**知识库构建**（进行中）：
- 三阶段构建流程：探测（kb_probe）→ 主题提取（kb_topics）→ 综合整理（kb_synthesize）
- 针对每个 Version 独立构建，同语言多版本知识库条目可跨版本共享
- 双重用途：前端层级展示给用户（小型语言 wiki），数据库供 LLM 检索（后期接入 embedding/rerank）

### 数据库

**迁移文件**：
- `server/migrations/001_initial_schema.sql` — languages + language_versions 表
- `server/migrations/002_add_language_icon.sql` — languages 添加 icon, source_urls 列
- `server/migrations/007_add_knowledge_unique.sql` — knowledge 表添加唯一约束
- `server/migrations/008_add_none_compatibility.sql` — compatibility_model 增加 'none'（无版本区分）

**Language 表字段**：id, name, slug, icon, compatibility_model, source_urls, created_at
**Knowledge 表**：按 language_version_id 组织，存储结构化知识条目（标题、内容、类型、层级路径）

### LLM 抽象层

- Provider 接口：`Chat(ctx, ChatRequest) (*ChatResponse, error)` + `Name() string`
- 纯 `net/http`，不引入 SDK，调用 `/chat/completions` 端点
- 多 provider + fallback 链 + token 用量跟踪（sync.Mutex）
- Prompt 模板：YAML 文件 + Go `text/template`，存储在 `server/prompts/`

### 当前 Prompt 文件

| 文件 | 用途 |
|------|------|
| `language_analyze.yaml` | 阶段一：LLM 基于 Wikipedia 数据判定编程语言分类 + 兼容模式 |
| `language_research.yaml` | 阶段二：LLM 调研最新稳定版本 + 环境来源 + 递归获取历史版本 |
| `language_describe.yaml` | 阶段三：LLM 生成语言描述、icon 描述、生态信息 |
| `kb_probe.yaml` | 知识库构建 — 探测目标 URL 的结构和内容范围 |
| `kb_topics.yaml` | 知识库构建 — 从探测结果提取结构化主题列表 |
| `kb_synthesize.yaml` | 知识库构建 — 综合整理最终知识条目 |

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
│   │   │   ├── language.go         # LanguageHandler (CRUD + Init 三阶段流水线)
│   │   │   ├── version.go          # VersionHandler
│   │   │   ├── execute.go          # ExecuteHandler (代码提交执行)
│   │   │   ├── config.go           # ConfigHandler (LLM 配置管理)
│   │   │   ├── response.go         # RespondJSON / RespondError
│   │   │   └── middleware.go       # Recovery, Logging, CORS
│   │   ├── service/
│   │   │   ├── language_service.go # Language CRUD
│   │   │   ├── language_init_service.go  # 三阶段初始化流水线（分析→调研→描述）
│   │   │   ├── version_service.go  # Version CRUD
│   │   │   ├── init_service.go     # 初始化流程编排
│   │   │   ├── kb_build_service.go # 知识库构建服务
│   │   │   └── kb_explorer.go      # 知识库内容探索
│   │   ├── model/
│   │   │   ├── language.go         # Language + LanguageVersion structs
│   │   │   └── knowledge.go        # Knowledge 知识条目 struct
│   │   ├── repo/
│   │   │   ├── db.go               # NewDB + RunMigrations
│   │   │   ├── language_repo.go    # LanguageRepo
│   │   │   ├── version_repo.go     # VersionRepo
│   │   │   └── knowledge_repo.go   # KnowledgeRepo
│   │   ├── llm/
│   │   │   ├── types.go            # Provider interface, ChatRequest/Response
│   │   │   ├── provider.go         # openAIProvider (OpenAI 兼容实现)
│   │   │   ├── service.go          # Service (多 provider + fallback + token 统计)
│   │   │   ├── template.go         # LoadTemplate (YAML + text/template)
│   │   │   ├── parse.go            # LLM 响应解析
│   │   │   └── template_test.go
│   │   ├── scraper/
│   │   │   ├── scraper.go          # Client (HTTP + Wikipedia base URL)
│   │   │   ├── wikipedia.go        # SearchWikipedia, GetPageCategories, GetInfobox, ScoreSignal
│   │   │   ├── compatibility.go    # CompatibilityModel heuristic (Go-level fast path)
│   │   │   └── fetcher.go          # FetchPageText (HTML → 纯文本)
│   │   ├── executor/
│   │   │   ├── executor.go         # 代码执行编排
│   │   │   └── runtime.go          # 运行时环境抽象
│   │   ├── docker/
│   │   │   └── client.go           # Docker 客户端（容器管理）
│   │   └── config/
│   │       ├── config.go           # Load, Save, LLMConfig, MaskKey, ToResponse
│   │       └── config_test.go
│   ├── prompts/
│   │   ├── language_analyze.yaml   # 阶段一：语言分类 + 兼容模式判定
│   │   ├── language_research.yaml  # 阶段二：版本 + 环境来源调研
│   │   ├── language_describe.yaml  # 阶段三：语言描述生成
│   │   ├── kb_probe.yaml           # 知识库构建 — 结构探测
│   │   ├── kb_topics.yaml          # 知识库构建 — 主题提取
│   │   └── kb_synthesize.yaml      # 知识库构建 — 综合整理
│   ├── migrations/
│   │   ├── 001_initial_schema.sql
│   │   ├── 002_add_language_icon.sql
│   │   ├── 007_add_knowledge_unique.sql
│   │   └── 008_add_none_compatibility.sql
│   └── config.yaml
├── web/
│   ├── src/
│   │   ├── api/client.ts           # 类型化 API 客户端
│   │   ├── components/Layout.tsx   # 暗色 sticky header + 导航
│   │   ├── pages/
│   │   │   ├── HomePage.tsx
│   │   │   ├── LanguagesPage.tsx
│   │   │   ├── LanguageDetailPage.tsx  # 知识库展示 + 嵌入式 Playground + 删除
│   │   │   ├── AddLanguagePage.tsx     # Wikipedia 验证 + 三阶段初始化确认
│   │   │   ├── SettingsPage.tsx        # LLM 配置管理
│   │   │   └── NotFoundPage.tsx
│   │   └── App.tsx
│   └── vite.config.ts              # Tailwind v4 plugin + /api proxy
└── CLAUDE.md
```

## 核心数据模型（已实现部分）

```
Language
  ├─ id, name, slug, icon
  ├─ compatibility_model: "strict" | "versioned" | "none"
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

- `compatibility_model`："strict" 语言一个活跃版本（Go, Java），向后兼容保证；"versioned" 语言多版本并存（Python, C++, Rust），各版本独立维护；部分语言无版本区分（如 SQL 方言）
- UserProject 锚定到 LanguageVersion
- 用户学习画像以 Language 为单元跨版本计算，同语言多版本之间部分进度共通、部分知识库共享
- 知识库双重用途：前端展示给用户（小型语言 wiki），数据库供 LLM 检索（后期接入 embedding/rerank 模型）

## 开发约定

- Go 三层架构：handler → service → repo，标准库风格，无重量框架
- 数据库：sqlx + 原生 SQL，不引入 ORM
- LLM：纯 `net/http` 调用 OpenAI 兼容 `/chat/completions`，不引入 SDK
- 提示词：YAML 模板 + Go `text/template`，温度 0.1，短小精悍
- 前端：React Context + useReducer，不引入 Redux
- API 响应：`{ok: bool, data?: T, error?: string}`
- Wikipedia User-Agent：`LearnCode/1.0`
- 执行安全：Docker 容器隔离（优先），`os/exec` + ulimit（回退），默认 30s 超时，无网络
- LLM 资源约束：默认面向 4-10B 本地部署 4-bit 量化模型，16k 上下文窗口，最大 4k 输出；每个 LLM 调用环节都需评估模型在此限制下能否胜任

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

按优先级排列：

1. **前端重构与问题修复**（当前进行中）
2. **Version 递归拓展** — Language 创建后自动获取所有受支持的历史版本（版本号 + 环境来源）
3. **容器运行时构建** — 依据环境来源为每个 Version 构建隔离容器，验证可用后标记就绪
4. **知识库构建** — 三阶段构建流水线完善（kb_probe → kb_topics → kb_synthesize），前端以小型语言 wiki 形式展示
5. **练习系统** — LLM 生成练习项目（需求文档 + 隐藏的参考答案与验证脚本），两步验证（验证脚本覆盖检查 → 参考答案可跑通检查），用户提交代码 → 容器验证 → 项目库纳入
6. **用户系统** — 学习画像、进度追踪、个人项目库
