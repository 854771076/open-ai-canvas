# 01 · 配置与环境变量面

本页从 `backend/cmd/server/main.go` 的 `run()` 装配、各业务包 `os.Getenv` 读取点与仓库 7 份 docker-compose 编排逐项枚举运行配置面；默认值以代码与编排字面为准。

## 进程入口配置链

`backend/cmd/server/main.go` `main()` → `logging.ConfigFromEnv`（`backend/internal/logging`）装配日志；`run()` 依次：读数据目录并 `MkdirAll` → `database.Open`（驱动/DSN/数据目录）→ `CANVAS_AUTO_MIGRATE` 决定启动迁移或 `database.RequireSchemaVersion` 校验版本 → 装配更新器客户端 → 组装 service → 建 Gin 与路由 → 监听 `CANVAS_BACKEND_ADDR` → 启动任务 Worker；收到 SIGTERM 后 HTTP 30 秒内排空，后台任务按 `CANVAS_SHUTDOWN_TIMEOUT`（`envDuration`）等待退出。

## 环境变量表

| 变量 | 作用 | 默认 | 出现位置（路径 + 符号） |
| --- | --- | --- | --- |
| CANVAS_BACKEND_DATA_DIR | 本地数据目录，启动时创建 | `data` | `backend/cmd/server/main.go` `run()`（`env("CANVAS_BACKEND_DATA_DIR", "data")`）；deploy/server 编排设 `/data` 并挂卷 |
| CANVAS_DATABASE_DRIVER | 数据库驱动 `sqlite` / `postgres` | `sqlite` | `backend/cmd/server/main.go` `run()`（传入 `database.Open`）；deploy 编排固定 `postgres` |
| DATABASE_URL | 数据库 DSN | 空 | `backend/cmd/server/main.go` `run()`（`os.Getenv`）；deploy 编排必填（`${DATABASE_URL:?…}`） |
| CANVAS_AUTO_MIGRATE | `true` 启动时迁移；`false` 仅校验 schema 版本 | `true` | `backend/cmd/server/main.go` `run()`（`envBool`）；deploy/server 编排设 `"false"`，由一次性 migrate 服务执行迁移 |
| CANVAS_SHUTDOWN_TIMEOUT | SIGTERM 后后台任务退出等待；compose `stop_grace_period` 同值 | `10m` | `backend/cmd/server/main.go` `run()`（`envDuration("CANVAS_SHUTDOWN_TIMEOUT", 10*time.Minute)`）；各生产编排 `stop_grace_period: ${CANVAS_SHUTDOWN_TIMEOUT:-10m}` |
| CANVAS_BACKEND_ADDR | HTTP 监听地址 | `:8080` | `backend/cmd/server/main.go` `run()`（`env("CANVAS_BACKEND_ADDR", ":8080")`）；全部编排设 `":8080"` |
| REDIS_URL | 多实例协调（限流/并发/熔断）；PG 多实例模式强制 | 空 | `backend/internal/platform/coordination.go`（PG 多实例未配置直接拒绝启动："PostgreSQL 多实例模式必须配置 REDIS_URL…"）；deploy 编排固定 `redis://redis:6379/0` |
| YINGCE_AGENT_URL | 远程 Agent 容器端点；非空走远程 HTTP，否则内嵌 spawn | 空 | `backend/internal/app/cloud_agent_pi_runtime.go` `runCloudAgentPi`；deploy 设 `http://yingce-agent:8081`；browser-only 叠加层置空 |
| YINGCE_AGENT_TOKEN | 远程 Agent 鉴权 Token | 空；deploy 回退 `CANVAS_UPDATER_TOKEN` | `runCloudAgentPi`；`docker-compose.deploy.yml` yingce-agent/backend 两服务 |
| YINGCE_AGENT_BRIDGE_HOST | Agent 回连 Bridge 的主机名 | 空 | `runCloudAgentPi`；deploy 设 `backend` |
| CANVAS_AGENT_BROWSER_ONLY | `true` 时拒绝服务端 Node Agent，仅浏览器 Worker | `false` | `backend/internal/platform/runtime_policy.go` `BrowserAgentOnly()`（精确等于 `"true"`）；`runCloudAgentPi` 入口先行拒绝；browser-only 编排置 `"true"` |
| CANVAS_PI_RUNTIME_DIR | 内嵌 Pi 运行时文件目录 | 回退候选 `/app/backend/agent-runtime/pi` | `backend/internal/agent/runtime/runtime.go`（candidates 首位读此变量；缺失报 "agent runtime files are missing; set CANVAS_PI_RUNTIME_DIR"） |
| CANVAS_UPDATER_SOCKET | 宿主更新器 Unix socket | `/run/open-ai-canvas-updater/updater.sock` | `backend/cmd/server/main.go` `run()`（`updaterclient.New`）；deploy 编排同名注入 |
| CANVAS_UPDATER_TOKEN | 更新器 Token；非空才装配更新客户端 | 空 | `backend/cmd/server/main.go` `run()`；deploy 编排必填，并作为 `YINGCE_AGENT_TOKEN` 回退值 |
| CANVAS_OFFICIAL_PLUGIN_DIR | 官方插件包目录 | 依次回退仓库 `plugin-packages` | `backend/internal/app/protocol_plugins.go`（配置无效报 "CANVAS_OFFICIAL_PLUGIN_DIR 不是可读目录"，找不到报 "未找到官方 plugin-packages 目录；请设置 CANVAS_OFFICIAL_PLUGIN_DIR"） |
| CANVAS_CORS_ORIGINS | 允许的跨域来源，逗号分隔，`*` 全放行 | 编排默认 `http://localhost:3000`（server 形态 `*`；deploy 留空 = 仅同源 + localhost） | `backend/cmd/server/main.go` `cors()`、`parseCORSPolicy` |
| CANVAS_ALLOW_PRIVATE_UPSTREAMS | 整体放开私网上游（SSRF 防护豁免） | `false` | `backend/internal/outbound/outbound.go` `allowPrivateUpstreams` |
| CANVAS_ALLOWED_PRIVATE_UPSTREAM_HOSTS | 私网上游精确主机白名单，逗号分隔 | 空 | `backend/internal/outbound/outbound.go` `AllowedPrivateUpstreamHost`；`backend/internal/storage/s3.go`（对象存储 Endpoint 仅允许此白名单主机） |
| CANVAS_HTTP_PORT | deploy 形态 web 宿主端口 | `3000` | `docker-compose.deploy.yml` web.ports（`"${CANVAS_HTTP_PORT:-3000}:3000"`） |
| CANVAS_BACKEND_IMAGE / CANVAS_WEB_IMAGE | GHCR 后端/web 镜像，必须含 digest | 必填（`${VAR:?…}`） | `docker-compose.deploy.yml` migrate、backend、web |
| CANVAS_YINGCE_AGENT_IMAGE / CANVAS_IMAGE_TAG | Agent 镜像（默认 `ghcr.io/ddcat-ai/open-ai-canvas-yingce-agent:${CANVAS_IMAGE_TAG}`） | `CANVAS_IMAGE_TAG` 必填 | `docker-compose.deploy.yml` yingce-agent |
| MAX_CONCURRENT_SESSIONS | Agent 会话并发上限 | `8`（硬顶 64，再与 SESSION_CAP 取小） | `yingce-agent/server.mjs`（会话上限计算）；deploy 以 `${YINGCE_AGENT_MAX_SESSIONS:-8}` 注入 |
| CANVAS_WORKER_CONCURRENCY / CANVAS_CHANNEL_CONCURRENCY | 后台任务与渠道并发 | `3` | 各编排 backend environment 透传（dev/local/server/deploy 同名） |
| CANVAS_REGISTRATION_ENABLED / CANVAS_BROWSER_AGENT_ENABLED | 公开注册、浏览器 Agent 编排开关 | `false` | `.env.example`；各编排透传 |
| CANVAS_OIDC_ISSUER / CLIENT_ID / CLIENT_SECRET / REDIRECT_URL / ALLOWED_GROUP | OIDC 登录（回调路径 `/api/auth/oidc/callback`） | 空（留空关闭） | `.env.example`；deploy 编排透传 |
| CANVAS_PUBLIC_BASE_URL | 签发上游可取短时链接的站点地址 | 空 | `.env.example`；dev/local/deploy 编排透传 |
| CANVAS_ADMIN_ASSISTANT_OAUTH_CLIENTS | 预注册 MCP OAuth 公开客户端 JSON 数组 | 空 | `.env.example`；deploy 编排透传 |
| CANVAS_MCP_ALLOWED_PRIVATE_HOSTS | MCP 独立私网白名单（不继承模型渠道放行项） | 空 | `.env.example`；deploy 编排透传 |
| CANVAS_DATA_PATH / CANVAS_WEB_DATA_PATH / CANVAS_POSTGRES_DATA_PATH / CANVAS_REDIS_DATA_PATH | 编排卷宿主路径 | `backend-data` / `web-data` / `postgres-data` / `redis-data` | `docker-compose.deploy.yml` volumes 段 |
| CANVAS_LOG_LEVEL / CANVAS_LOG_FORMAT | 日志级别与格式 | `info` / `text` | `backend/internal/logging` `ConfigFromEnv`（`main()` 调用）；编排透传 |
| GIN_MODE | Gin 运行模式 | 生产编排设 `release`，dev 为 `debug` | 各编排 backend environment |
| POSTGRES_DB / POSTGRES_USER / POSTGRES_PASSWORD | PG 容器初始化 | `open_ai_canvas` / `open_ai_canvas` / 必填 | `docker-compose.deploy.yml`、`docker-compose.server.yml` postgres |
| YINGCE_AGENT_SESSION_CAP / NODE_MEMORY_MB / MAX_QUEUE / QUEUE_TIMEOUT_SECONDS / BODY_TIMEOUT_SECONDS / KILL_GRACE_SECONDS / ORPHAN_TTL_SECONDS / TMPFS_SIZE / MEMORY_LIMIT / CPUS / PIDS_LIMIT | Agent 资源边界 | 见 `docker-compose.deploy.yml` 缺省 | `docker-compose.deploy.yml` yingce-agent + `.env.example` |
| ENABLE_PROVIDER_PLUGINS | 服务商插件机制开关 | `false` | `docker-compose.local.yml`、`.env.example` |
| VITE_API_PROXY_TARGET / VITE_TLDRAW_LICENSE_KEY | dev 前端 API 代理目标 / tldraw 授权 key | `http://backend:8080` / 空 | `docker-compose.dev.yml` web；`docker-compose.build.yml` web build args |

## compose 形态差异对照

| 维度 | `docker-compose.dev.yml` | `docker-compose.yml`（基础） | `docker-compose.local.yml` | `docker-compose.deploy.yml` | `docker-compose.server.yml` |
| --- | --- | --- | --- | --- | --- |
| 镜像来源 | 现场构建：backend 用 `backend/Dockerfile.dev`，web 用 `oven/bun:1.3.13` 直跑 `bun install --frozen-lockfile && bun run dev` | 本地构建 `open-ai-canvas-backend:local` / `open-ai-canvas-web:local`（`backend/Dockerfile` + 根 `Dockerfile`） | 同基础，另加 `develop.watch` | GHCR digest：`CANVAS_BACKEND_IMAGE` / `CANVAS_WEB_IMAGE` 必填含 digest，Agent 默认 `ghcr.io/ddcat-ai/open-ai-canvas-yingce-agent:${CANVAS_IMAGE_TAG}` | 源码构建 `open-ai-canvas-backend:server` 等（`backend/Dockerfile` + 根 `Dockerfile`） |
| 数据库 | 默认 `sqlite`（`DATABASE_URL` 留空），无 PG/Redis 服务 | `sqlite` | `sqlite` | `postgres:17-alpine` + `redis:7.4-alpine`（AOF everysec）+ 一次性 `migrate` 服务（entrypoint `migrate-schema up`，backend `depends_on: service_completed_successfully`） | 同 deploy 的 PG+Redis+migrate 三件套，镜像为本地构建 |
| yingce-agent | 无此服务 | 无 | 无 | 独立服务：`read_only: true` + tmpfs `/tmp` + `cap_drop: [ALL]` + `no-new-privileges` + mem/cpus/pids 限额 + `127.0.0.1:8081/healthz` 健康检查 | 无 agent 服务 |
| backend 端口 | `127.0.0.1:${CANVAS_BACKEND_HOST_PORT:-8080}:8080` | 无宿主端口（经 web 反代） | 同基础 | 无宿主 ports | 无宿主 ports |
| web 端口/卷 | `${CANVAS_WEB_HOST_PORT:-3000}:3000`，bind 源码 + node_modules/Bun 缓存卷 | `3000:3000`，仅 `backend-data` 卷 | `3000:3000` + `web-data` 卷 | `${CANVAS_HTTP_PORT:-3000}:3000` + `web-data` 卷（注释：保留新旧两版 hashed chunks 供旧标签页） | `127.0.0.1:5173:3000` |
| 源码与缓存 | bind 挂 `. `→`/workspace`；`GOCACHE`/`GOMODCACHE` bind `./.local/cache/go-build|go-mod`；数据 bind `./.local/project-workbench-debug`（`create_host_path: false` 防误建） | `backend-data` 命名卷 | `backend-data` + `web-data` 卷 + `develop.watch` rebuild（backend 看 `./backend`，web 看 `./web`、`Dockerfile`、`nginx.conf`、`VERSION`、`CHANGELOG.md`） | 命名卷四件套 + updater socket bind `/run/open-ai-canvas-updater` | 宿主路径绑定（`./data`、`./postgres-data`、`./redis-data`），带 `container_name` |
| CORS/日志默认 | `CANVAS_CORS_ORIGINS` 默认 `http://localhost:3000`；`GIN_MODE=debug`；`CANVAS_LOG_LEVEL=debug` | 同左 CORS；`release`/`info` | 同基础 + `ENABLE_PROVIDER_PLUGINS` 透传 | `CANVAS_CORS_ORIGINS` 留空；`release`/`info` | `CANVAS_CORS_ORIGINS` 默认 `*`；`release`/`info` |

叠加层（与 deploy 组合使用）：

- `docker-compose.browser-only.yml`：`yingce-agent.profiles: [server-agent]` 隐藏 Agent 服务；backend 覆写 `CANVAS_AGENT_BROWSER_ONLY: "true"`、`YINGCE_AGENT_URL: ""`，并以 `depends_on.yingce-agent: !reset null` 解除依赖；文件头注释说明保留 Go 任务 Worker、停用 Node Agent 服务。`scripts/install-server-image.sh` 以 `BROWSER_COMPOSE_FILE` 引用该文件实现自动叠加。
- `docker-compose.build.yml`：为 migrate、yingce-agent、backend、web 四服务补 build 段，镜像统一 `:server` tag（`open-ai-canvas-backend:server`、`open-ai-canvas-yingce-agent:server`、`open-ai-canvas-web:server`），web 额外传入 `VITE_TLDRAW_LICENSE_KEY` build arg。
- `docker-compose.server.yml`：源码构建 + PostgreSQL 的自托管编排，无 agent；仓库脚本与 CI 中均无对该文件的引用（grep `scripts`、`.github` 无命中）。

## Agent 双模式配置关系

判定收敛在 `backend/internal/app/cloud_agent_pi_runtime.go` `runCloudAgentPi`，互斥顺序如下：

1. 先验 `CANVAS_AGENT_BROWSER_ONLY`：经 `backend/internal/platform/runtime_policy.go` `BrowserAgentOnly()` 判定为 `true` 时直接返回错误「当前部署仅支持浏览器 Worker，服务端 Agent 已停用」，不再进入任何服务端执行路径。
2. 再验 `YINGCE_AGENT_URL`：非空则 `yingceagent.Run` 走远程 HTTP 容器（`backend/internal/agent/yingceagent`），携带 `YINGCE_AGENT_TOKEN` 与 `YINGCE_AGENT_BRIDGE_HOST`；这是 deploy 编排的形态（`http://yingce-agent:8081` + `backend`）。
3. 两者皆未命中则 `agentruntime.Run` 内嵌 spawn（`backend/internal/agent/runtime`），运行时文件按 `CANVAS_PI_RUNTIME_DIR`（候选 `/app/backend/agent-runtime/pi`）解析；dev/local/base 编排未设远程变量，落在这一分支。

browser-only 叠加层构成双重互斥：既把 `YINGCE_AGENT_URL` 置空关闭远程容器，又置 `CANVAS_AGENT_BROWSER_ONLY=true` 拒绝内嵌 spawn，仅浏览器 Worker 与 Go 任务 Worker 存续；Agent 会话并发由 `MAX_CONCURRENT_SESSIONS` 约束（`yingce-agent/server.mjs`，默认 8、硬顶 64，再与 `YINGCE_AGENT_SESSION_CAP` 取小）。
