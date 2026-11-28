# 01 · HTTP 路由与端点清单

本页从 `backend/internal/handler/api.go` 的 `RegisterCanvasAPI` 注册全集与 `backend/cmd/server/main.go` 的 `run()` 装配代码逐项枚举路由域、流式端点与中间件链，所有条目均来自代码实读。端点级全量契约以 `backend/internal/handler/openapi.yaml` 为准，本页为域级导航与流式/中间件细节，未展开的域见「路由域清单」文末补遗。

## 路由总入口

- `backend/cmd/server/main.go` `run()`：`gin.New()` 之后挂 `/api` 组，按序注册 `registerSystemStatusRoutes`（健康检查）、`RegisterOAuthCallbackRoutes`（挂在引擎根，不在 `/api` 下）、`RegisterAdminAssistantMetadata`（引擎根 `/.well-known`）、`RegisterCanvasAPI(api, svc)`，最后 `r.NoRoute(handler.SystemProxyNoRouteHandler(svc))` 兜底。
- `backend/internal/handler/api.go` `RegisterCanvasAPI`：路由注册总入口（76 行），第一行 `api.Use(StorageReadOnlyGuard(svc))` 为存储只读兜底守卫，注释明示必须先于所有业务路由挂载；随后调用 40 余个 `Register*` 域函数；前后端同仓同发，不做路径版本前缀。
- `backend/internal/handler/routes.go`（363 行）承载任务域细节：`RegisterTaskRoutes` 下分 `registerTaskSubmissionRoutes`、`registerTaskReadRoutes`、`registerTaskTextReadRoutes`、`registerPersonalTaskActions`。
- 下表路径模式均省略 `/api` 前缀；仅 auth 域的 OAuth 回调与 `.well-known` 两处为引擎根路径，单列说明。

## 路由域清单

| 域 | 路径模式（省略 `/api` 前缀） | 代表注册符号 |
| --- | --- | --- |
| auth（登录/注册/OAuth 回调） | `/auth/settings`、`POST /auth/register`、`POST /auth/login`、`POST /auth/logout`、`GET /auth/session`、`POST /auth/email-code`、`POST /auth/password-reset-code`、`POST /auth/password-reset`、`/auth/linuxdo/*`、`/auth/oidc/*`、`/auth/wechat/*`、`/auth/profile/wechat/*`、`/auth/notification-settings`、`/auth/verification*`；引擎根 `GET /oauth/linuxdo/callback` | `backend/internal/handler/auth.go` `RegisterAuthRoutes`、`RegisterOAuthCallbackRoutes` |
| tasks（提交 + SSE） | `POST /tasks`（团队空间另有 `POST /tasks/quote`）、`GET /tasks`、`GET /tasks/:id`、`POST /tasks/:id/text-deltas`、`GET /tasks/:id/text-deltas`、`GET /tasks/:id/text-events`、`POST /tasks/:id/retry|recover-media|query-provider|cancel|text-replay-complete`、`GET /tasks/:id/logs`、`GET /admin/text-replay-stats` | `backend/internal/handler/routes.go` `RegisterTaskRoutes`、`registerTaskSubmissionRoutes`、`registerTaskTextReadRoutes` |
| resources（上传/公开分发） | `POST /resources/uploads`、`PUT /resources/uploads/:id/chunks/:index`、`POST /resources/uploads/:id/complete`、`GET /resources`、`POST /resources`、`POST /resources/import`、`GET /resources/:id/file`、`GET /resources/storage-usage`、`GET /resources/storage-grants`、`GET /public/resources/:id/file`、`GET /public/resources/:id/file/:filename` | `backend/internal/handler/resource_upload_session.go` `RegisterChunkedUploadRoutes`；`backend/internal/handler/user_data.go` `RegisterUserDataRoutes`；`backend/internal/handler/storage_grants.go` `RegisterStorageGrantRoutes` |
| canvas（文档/实时） | `GET /canvas-projects`、`GET /canvas-projects/:id`、`GET /canvas-projects/:id/sync-status`、`POST /canvas-projects/:id/edit-lease`、`PATCH /canvas-projects/:id/operations`、`DELETE /canvas-projects/:id`、`GET /canvas-projects/:id/realtime`（WebSocket）、`GET|POST|DELETE /canvas-projects/:id/share`、`GET /public/canvas-shares/:token`、`GET /public/canvas-shares/:token/resources/:resourceId/file`、`POST /canvas-projects/:id/import/libtv`、`POST /canvas-projects/:id/import/tapnow` | `backend/internal/handler/canvas_realtime.go` `RegisterCanvasRealtimeRoutes`；`backend/internal/handler/canvas_share.go` `RegisterCanvasShareRoutes`；`backend/internal/handler/user_data.go` `RegisterUserDataRoutes` |
| agent | `GET /agent/capabilities`、`GET|DELETE /agent/conversations/:id`、`GET /agent/conversations/:id/runs`、`POST /agent/conversations/:id/compact`、`POST /agent/runs`、`POST /agent/runs/:id/messages|interjections|cancel|undo`、`GET /agent/runs/:id/events`（SSE）、`POST /agent/runs/:id/approvals/:approvalId/decision`、`GET|PATCH /agent/profile`、`/agent/memories*`、`/agent/memory*` | `backend/internal/handler/agent.go` `RegisterAgentRoutes`；`backend/internal/handler/agent_lesson.go` `RegisterAgentLessonAdminRoutes`；`backend/internal/handler/agent_memory_document.go` `RegisterAgentMemoryRoutes` |
| plugins | `GET /plugins/status`、`GET /plugins/catalog`、`GET /plugins/:id/package`、`POST /plugins/:id/enable|disable`、`PUT /plugins/:id/activation`、`DELETE /plugins/:id`、`GET|POST /plugins/eagle/*`；`GET|PUT|DELETE /admin/plugins/*` | `backend/internal/handler/plugin.go` `RegisterPluginRoutes`（`/plugins` 与 `/admin/plugins` 双分组） |
| skills/prompts | `GET /skills`、`GET /skills/:id`、`GET /skills/:id/files|file|file/raw|bundle|search`、`POST|DELETE /skills/:id/add|like`、`PATCH /skills/:id/agent-mode`、`GET /skills/library-categories|presets|recommendations|added`、`GET|PATCH|DELETE /settings/prompt-templates*`；`POST /admin/skills/install`、`POST /admin/skills/install/github`、`GET|PUT|DELETE /admin/skills/*`、`GET|POST|PATCH|DELETE /admin/prompt-templates*` | `backend/internal/handler/skills.go` `RegisterSkillRoutes`；`backend/internal/handler/user_data.go` `RegisterUserDataRoutes`；`backend/internal/handler/admin_skill_routes.go` `RegisterAdminSkillRoutes`；`backend/internal/handler/admin_channel_routes.go`（prompt-templates 段） |
| admin 后台 | `/admin/users*`、`/admin/user-groups*`、`/admin/channels*`、`/admin/provider-catalog*`、`/admin/settings/*`、`/admin/api-logs*`、`/admin/login-logs*`、`/admin/analytics/*`、`/admin/model-pricings*`、`/admin/storage/*`、`/admin/resources/*`、`/admin/system-update*`、`/admin/system-performance*`、`/admin/agent-lessons*`、`/admin/references` | `backend/internal/handler/auth.go` `RegisterAdminRoutes`；`backend/internal/handler/admin_analytics.go` `RegisterAdminAnalyticsRoutes`；`backend/internal/handler/admin_update.go` `RegisterAdminUpdateRoutes`；`backend/internal/handler/admin_system_performance.go` `RegisterAdminSystemPerformanceRoutes`；`backend/internal/handler/admin_storage.go` `RegisterAdminStorageRoutes` |
| payments/wallet | `GET /wallet`、`POST /wallet/redeem`、`POST /wallet/checkin`、`GET /payments/providers|products|orders`、`POST /payments/orders`、`GET /payments/orders/:id`、`POST /payments/orders/:id/query` | `backend/internal/handler/finance.go` `RegisterFinanceRoutes`；`backend/internal/handler/payment.go` `RegisterPaymentRoutes`；`RegisterReferralRoutes`（api.go 同批挂载） |
| teams | `GET|POST /teams`、`POST /teams/:teamId/groups`、`GET /teams/:teamId/members|invite-candidates|invitations`、`POST /teams/:teamId/invitations`、`POST /teams/:teamId/invitations/:id/revoke`、`POST /teams/:teamId/leave`、`DELETE /teams/:teamId/members/:membershipId`、`GET /team-invitations`、`POST /team-invitations/:id/accept|decline`、`/teams/:teamId/workspace/*`（复用任务/上传/画布实时注册器） | `backend/internal/handler/team.go` `RegisterTeamWorkspaceRoutes`、`registerTeamRoutes` |
| system proxy | `ANY /ai/system/:channelId/*path`；NoRoute 短路径 `/api/{channelId}/{providerPath}`；`ANY /ai/custom` | `backend/internal/handler/auth.go` `RegisterSystemProxyRoutes`、`SystemProxyNoRouteHandler`；`backend/internal/handler/custom_proxy.go` `RegisterCustomRelayRoutes` |
| system/ops | `GET /health/live`、`GET /health/startup`、`GET /health/ready`、`GET /health`、`GET /system/version`、`GET /openapi.yaml`、`POST /diagnostics/preview|export`、`GET /public/appearance*`、`GET /public/welcome` | `backend/cmd/server/system_status.go` `registerSystemStatusRoutes`；`backend/internal/handler/api.go` `RegisterOpenAPIRoutes`；`backend/internal/handler/diagnostics.go` `RegisterDiagnosticsRoutes` |

api.go 全集补遗（同批注册、上表未展开）：`RegisterAppearanceRoutes`、`RegisterFeatureAvailabilityRoutes`、`RegisterMCPRoutes`、`RegisterAdminAssistantRoutes`（`/admin-assistant` 分组，MCP 与 OAuth 端点，见 `backend/internal/handler/admin_assistant.go`；其 `/.well-known/oauth-*` 元数据挂引擎根）、`RegisterResponseInterceptionRoutes`、`RegisterOfficialAssetRoutes`、`RegisterOfficialAssetRecommendationRoutes`、`RegisterSiteTrafficRoutes`、`RegisterAnnouncementRoutes`、`RegisterCreationRoutes`、`RegisterLibTVRoutes`、`RegisterTapNowRoutes`、`RegisterChannelModelRoutes`、`RegisterLogicalModelRoutes`、`RegisterModelCatalogRoutes`、`RegisterEcommerceRoutes`、`RegisterEcommerceAgentRoutes`（`/canvas/placeholder|save`、`GET /canvas/:projectId`）、`RegisterRunningHubRoutes`、`RegisterAdminSkillRoutes`、`RegisterToolRoutes`、`RegisterPoseLibraryRoutes`、`RegisterInspirationFeedRoutes`、`RegisterWorkbenchRoutes`、`RegisterLibraryRoutes`、`RegisterFeedbackRoutes`、`RegisterFeishuFeedbackRoutes`、`RegisterArkPortraitRoutes`、`RegisterProjectRoutes`（经 `RequireFeature(svc, service.FeatureShortDrama)` 门控）、`RegisterCanvasShareRoutes`。

## SSE 与 WebSocket 端点

### 任务文本事件流

- 端点：`GET /api/tasks/:id/text-events`，团队空间镜像 `GET /api/teams/:teamId/workspace/tasks/:id/text-events`（`backend/internal/handler/team.go` `RegisterTeamWorkspaceRoutes` 复用 `registerTaskTextReadRoutes`）。
- 注册符号：`backend/internal/handler/routes.go` `registerTaskTextReadRoutes`（路由字面量含 `text-events`），流循环 `streamTaskTextEvents`、游标解析 `taskTextEventCursor`、事件写出 `writeTaskTextSSE`。
- 形态：GET + `text/event-stream`；游标双通道，查询参数 `after` 与请求头 `Last-Event-ID` 同时可带、取较大者；事件 `id` 取 delta sequence，事件类型 `progress`、`delta`、`terminal`、`error`，另有 `: heartbeat` 注释行；750ms 轮询上游、15s 心跳；响应头 `X-Accel-Buffering: no` 防代理缓冲。

### Agent 事件流

- 端点：`GET /api/agent/runs/:id/events`，见 `backend/internal/handler/agent.go` `RegisterAgentRoutes`。
- 形态：游标即事件 seq，同样经 `taskTextEventCursor` 解析 `after`/`Last-Event-ID`；`agent_event` 事件带 `id=seq` 推进游标，`run_snapshot` 快照刻意不带 `id`、不推进游标；运行进入终态（completed/failed/cancelled/rejected）后关流；15 秒无事件写 `: heartbeat`。

### 画布实时（WebSocket Upgrade）

- 端点：`GET /api/canvas-projects/:id/realtime`，见 `backend/internal/handler/canvas_realtime.go` `RegisterCanvasRealtimeRoutes`；经 `websocket.Accept`（coder/websocket）完成 Upgrade。
- 形态：`welcome`/`presence`/`heartbeat` JSON 帧；上行仅接受 presence 消息且限频；15s Ping 复核登录会话与画布可用性；团队空间镜像 `/api/teams/:teamId/workspace/canvas-projects/:id/realtime`（`RegisterTeamWorkspaceRoutes` 内复用同一注册器）。
- Origin 校验复用共享 CORS 中间件的结果：仅当响应已写 `Access-Control-Allow-Origin` 等于请求 Origin 时放行该 Origin。

### 系统代理流式透传

- 端点：`ANY /api/ai/system/:channelId/*path`（`RegisterSystemProxyRoutes`）与 NoRoute 短路径（`SystemProxyNoRouteHandler`）。
- 形态：`backend/internal/handler/auth.go` `proxySystemRequestPath` 内以 `isSystemProxyEventStream` 判定上游为 SSE 后走 `streamSystemProxyResponse` 逐块回写，覆盖 OpenAI/Gemini 型流式响应；Token 计费的 Chat Completions 请求注入 `stream_options.include_usage`（`EnsureChatCompletionStreamUsageRequest`），被严格 schema 上游拒收时按浏览器原始请求体重发一次（`resendIfStreamUsageRejected`）。

## 中间件链顺序

1. `logging.AccessLog(redactCanvasSharePath)` → `logging.Recovery()`：`backend/internal/logging/http.go` `AccessLog`、`Recovery`，在 `backend/cmd/server/main.go` `run()` 挂载；画布分享公开路径在访问日志中脱敏。
2. `handler.RequestCorrelationMiddleware()`：`backend/internal/handler/request-context.go`，生成/透传请求关联标识。
3. CORS：`backend/cmd/server/main.go` `cors()`，读 `CANVAS_CORS_ORIGINS`（`parseCORSPolicy`），未配置时仅放行同源与 localhost；WebSocket 端点因此无需独立 Origin 白名单。
4. `/api` 组内首道守卫：`StorageReadOnlyGuard`（`backend/internal/handler/api.go` `RegisterCanvasAPI`，置于全部业务路由之前）。
5. NoRoute 兜底：`handler.SystemProxyNoRouteHandler`（`backend/internal/handler/auth.go`），只处理短公开代理路径 `/api/{channelId}/{providerPath}`，刻意走 NoRoute 以免遮蔽 `/api/tasks`、`/api/plugins` 等既有业务路由。
6. OpenAPI：`GET /api/openapi.yaml`（`RegisterOpenAPIRoutes`，直接内嵌返回规格数据）。
7. 健康检查：`/api/health/live`、`/api/health/startup`、`/api/health/ready`、`/api/health`、`/api/system/version`（`backend/cmd/server/system_status.go` `registerSystemStatusRoutes`，ready 汇总 database/runtime/schema 三项检查）。

补充：中间件经 `r.Use` 全局施加，而 `RegisterOpenAPIRoutes`、`registerSystemStatusRoutes` 与各业务域注册都发生在全局中间件之后，符合 Gin「中间件只作用于其后注册路由」的语义（api.go 首行注释同述）。
