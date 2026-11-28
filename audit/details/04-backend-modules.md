# 后端模块细节卡（T01–T21）

本文件是后端模块编号表的细节增量：每张卡只补符号、装配顺序与边界事实，全部数字实测于当前代码树。

## 通用装配顺序（先读这张）

`backend/cmd/server/main.go` 起服务后，领域装配收敛在 `backend/internal/app/service.go` 的 `newService` 单函数里，顺序固定：`platform.NewCoordinator(repo.Dialect())` → `canvas.NewRealtimeHub(coordinator.Redis())` → `repo.WithCanvasChangeNotifier(canvasRealtime.Changed)` → `newPluginRuntime(dataDir)` → `payment.NewRegistry()`（插件运行时存在动态注册表快照时覆盖静态注册表）→ 构造 `Service` 字面量 → 五个任务协调器（`newTaskBillingCoordinator`/`newTaskTerminalCoordinator`/`newTaskRouteExecutor`/`newTaskWorkerCoordinator`/`newTaskLifecycleCoordinator`）→ 领域服务逐个挂载：`skills.New` → `tools.New` → `prompts.New` → `auth.New` → `newSMSDomain` + `auth.SetSMSDelivery` → `canvas.New` → `workbench.New` → `poselibrary.New` → `inspirations.New` → 最后 `platform.New`。任何域要新增协调器，改动点只有这一个函数；跳过它会丢掉实时通知或支付插件注入。

## T01 kernel

- **职责**：全仓无业务语义的基底工具：ID、错误类型、token 计费常量。
- **关键符号**：`backend/internal/kernel/util.go` 的 `NewID`（16 字节 crypto/rand 转十六进制，rand 失败才退化为纳秒时间戳）、`FirstNonEmpty`、`UniqueNonEmpty`、`DefaultString`；`backend/internal/kernel/errors.go`；`backend/internal/kernel/token_billing.go`。
- **依赖谁**：仅标准库（`crypto/rand`、`encoding/hex`、`fmt`、`strings`、`time`），不 import 仓内任何包。
- **被谁依赖**：所有内部包（`assets/access.go` 的 `AccessError` 直接构造 `kernel.AppError`，`canvas/realtime.go` 用 `kernel.NewID` 生成 hub 实例 ID）。
- **边界与观察**：`NewID` 的时间戳回退路径是唯一非随机分支，出现即意味着系统熵源故障，值得告警而非静默。

## T02 database

- **职责**：GORM 打开与 schema 版本推进。
- **关键符号**：`backend/internal/database/database.go`（默认 SQLite DSN `open_ai_canvas.db?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on&_synchronous=NORMAL`；PG 走 `pgx/v5/stdlib`；连接池实测 PG `MaxOpenConns=30`/`MaxIdleConns=10`，SQLite `8`/`4`）；`backend/internal/database/migrations.go` 的 `CurrentSchemaVersion`（实测 111）、迁移项结构体带 `checksum` 字段（迁移表逐条记录 sha256 校验和，防手改历史迁移）、PG 分支执行 `pg_advisory_xact_lock(postgresSchemaMigrationLockID)` 防多实例并发迁移、`RequireSchemaVersion`（业务侧启动自检，版本不满足直接拒绝服务）。
- **依赖谁**：GORM、pgx、`internal/logging`。
- **被谁依赖**：T04 repository、T21 全部 cmd 入口（`migrate-schema` 只做迁移，`server` 启动前调用 `RequireSchemaVersion`）。
- **边界与观察**：SQLite 连接池上限低于 PG 是刻意为之（WAL 单写者）；迁移 checksum 命中失败是硬错误，不允许跳过。

## T03 model

- **职责**：GORM 实体与领域值类型，全部持久化形状的唯一来源。
- **关键符号**：包内实测 212 个 struct 类型，分布在 `backend/internal/model/models.go` 与约 60 个领域文件：`models_task.go`、`models_finance.go`、`models_channel.go`、`models_logical_model.go`、`cloud_agent.go`、`models_canvas_sync.go`、`models_payment.go`、`models_platform.go` 等。
- **依赖谁**：GORM 标签，无内部依赖。
- **被谁依赖**：T04、T06–T18 全部业务包；迁移 baseline（T02 的 v1 `baseline_gorm_schema`）由这些实体的 AutoMigrate 形状生成。
- **边界与观察**：文件命名双轨并存——老 `models_*.go` 前缀族与后来的单实体文件（如 `cloud_agent.go`、`upload_reservation.go`）；找实体先看单实体文件再回退前缀族。

## T04 repository

- **职责**：唯一 SQL 触点；任务认领租约、计费事务。
- **关键符号**：`backend/internal/repository/repository_tasks.go` 的 `ClaimNextTask`——PG 分支用 `clause.Locking{Strength:"UPDATE", Options:"SKIP LOCKED"}` 锁行跳过竞争，SQLite 分支用同条件谓词做条件更新，两分支共享 `lease_owner`/`lease_expires_at` 租约语义；配套 `RenewTaskLease`（租约校验失败报「任务租约已失效」）、`ClaimFailedTaskProviderRecovery`（人工恢复只锁失败任务）；`backend/internal/repository/finance.go` 的 `CreateTaskWithCreditReservation`（任务行 + 积分预留单同事务提交，带 `activeTaskLimit` 并发准入）；`backend/internal/repository/finance_billing.go` 的 `SettleBillingOrder`、`SettleBillingOrderWithAudioDuration`、`RefundBillingOrder`、`AutoRefundBillingOrder`、`MarkBillingUncertain`。
- **依赖谁**：T02、T03。
- **被谁依赖**：T06 platform、T18 app（全部业务写入经它，app 层无裸 GORM 写）。
- **边界与观察**：文件头注释明确「执行中」写入全部带租约校验（`taskLeaseWriter`）：旧执行者被接管后的迟到写入因租约不匹配被丢弃，防止覆盖新结果或重复扣费；`MarkBillingUncertain` 刻意保留 uncertain 终态订单不自动结算，等待人工/自动退款裁决。

## T05 outbound

- **职责**：一切服务端出站 URL 的 SSRF 防线。
- **关键符号**：`backend/internal/outbound/outbound.go` 的 `ValidateOutboundURL`、`ValidateOutboundURLWithProxy`（代理场景保留 URL/字面 IP 校验、域名解析交给代理侧；`allowPrivateUpstreams`/`AllowedPrivateUpstreamHost` 提供白名单豁免）、`ValidateProxyURL`（代理本身允许认证信息与私网地址）；IP 判定集为 `IsLoopback || IsPrivate || IsLinkLocalMulticast || IsLinkLocalUnicast || IsUnspecified || IsMulticast`，命中即「不允许访问保留地址或特殊用途地址」；`backend/internal/outbound/mcp.go` 的 `ValidateMCPURL` 与 `allowedPrivateMCPHost`（私网 MCP 主机仅可经 `CANVAS_MCP_ALLOWED_PRIVATE_HOSTS` 环境变量逐个放行）。
- **依赖谁**：标准库 net/url。
- **被谁依赖**：T14 storage（`objects.go` import 它）、T08/T18 的上游请求构造、`internal/outbound/outboundtest` 测试基建。
- **边界与观察**：文件头策略——私网默认拒绝，但「渠道级代理」例外：代理地址允许私网与认证，因为代理本身是管理员配置的信任边界。

## T06 platform

- **职责**：跨实例协调（限流/并发槽/熔断）、worker 运行时、有界读缓存。
- **关键符号**：`backend/internal/platform/coordination.go` 的 `NewCoordinator(dialect)`——PG 方言且未配 `REDIS_URL` 时直接报错「PostgreSQL 多实例模式必须配置 REDIS_URL，用于限流、并发和熔断协调」；`fixedWindowScript`（`redis.NewScript`，键前缀 `canvas:rate:`，传窗口毫秒与 limit）；`defaultChannelConcurrencyValue = 8`（并发槽默认值实测）；配套 `NewCoordinatorWithRedis`、`HasRedis`、`Redis`。`backend/internal/platform/worker_runtime.go` 的 `Worker`（`Start`/`GoLoop`/`GoTask`/`BeginDrain`/`Stop`/`IsDraining`/`ActiveTaskCount`，drain 语义供优雅停机）。`backend/internal/platform/read_cache.go` 的 `BoundedReadCache[K,V]`（`Get(ctx,key,load)` + 条目数/字节数/加载次数三重上限 + `evict`）。
- **依赖谁**：go-redis、T02。
- **被谁依赖**：T18 app（`Service.coordinator`、`Service.workers`、三个 `BoundedReadCache` 字段均出自此包）。
- **边界与观察**：SQLite 单实例模式下 Coordinator 无 Redis 也能工作（`HasRedis=false` 时限流降级为进程内）；Redis 仅在 PG 多实例为强制项。

## T07 protocol

- **职责**：适配器接口、插件 manifest 装载、内置协议适配器。
- **关键符号**：`backend/internal/protocol/adapter.go` 的 `Adapter` 接口族（`BuildCreate`/`ParseCreate`/`BuildPoll`/`ParsePoll`/`BuildCancel`，另有 `AgentAdapter`、`ResultAdapter`、`MediaUploadAdapter` 能力接口与 `UnavailableAdapter` 兜底实现）；`backend/internal/protocol/manifest.go` 的 `ManifestOperation`/`ManifestUpload`/`ManifestFilePart`/`ManifestResponse`/`ManifestAgentResponse` 与 `LoadInstalledPlugin`/`LoadInstalledProviders`；`manifest_adapter.go`、`manifest_request.go`、`manifest_text.go`；`backend/internal/protocol/builtin.go` 的 `builtinAdapter`、`Builtins()` 注册表（`builtinRegistry`+互斥锁）、内置适配器工厂 `openAIChatAdapter`/`openAIResponsesAdapter`/`claudeAdapter`/`openAIVideosAdapter`/`newAPIVideosAdapter`；`backend/internal/protocol/package.go` 的 `PluginPackageFormat="yingce.plugin/package-v1"`、`PluginPackageMaxBytes=48MiB`、`pluginPackageMaxFiles=256`、`pluginPackageMaxEntry=16MiB`。
- **依赖谁**：标准库 + zip。
- **被谁依赖**：T08 provider、T18 app（`provider_protocol.go` 声明式宿主消费 manifest）。
- **边界与观察**：`DeclaredPostPaths` 不在本包——它是 `backend/internal/app/system_proxy_protocol.go` 里 `SystemProxyContract` 的方法，被 `backend/internal/handler/auth.go` 的系统代理授权调用；manifest 是能力契约、文件是可选附件（`manifest.entry` 寻址），二者解耦。

## T08 provider

- **职责**：模型目录插件接口与按 baseURL 探测的厂商目录补全。
- **关键符号**：`backend/internal/provider/interface.go` 的 `ModelCatalogPlugin`（注释：只补充标准 `/models` 没有暴露的厂商目录项）；`backend/internal/provider/bailian/discovery.go` 的 `Discovery.Match(baseURL, headers)`（解析 baseURL 判定是否百炼目录）；`backend/internal/provider/registry.go`。
- **依赖谁**：标准库 net/url。
- **被谁依赖**：T18 app 的 `providerCatalog`（`providercatalog.Source` 装配插件）。
- **边界与观察**：目录插件是「增补」语义，不允许覆盖标准目录；`bailian` 是唯一内置 Discovery 实现，新厂商照此模式加子目录。

## T09 assets

- **职责**：画布/文档内资源引用解析、访问意图、URL 形状。
- **关键符号**：`backend/internal/assets/references.go` 的 `DocumentReferences`/`DocumentReferencedIDs`/`CollectOwnedDocumentReferences`/`CollectDocumentResourceReferences`（递归 `walkReferenceDocument`，识别裸资源 ID 字段与资源定位符字段两类引用形态）；`backend/internal/assets/access.go` 的 `AccessPurpose`/`ResourceVariant`/`DeliveryMode` 枚举与 `AccessRequest`/`ResourceAccess`/`AccessError`；`backend/internal/assets/urls.go` 的 `FileURL`/`IsFileURL`/`IDFromFileURL`/`ResourceID`/`ValidID`。
- **依赖谁**：T01 kernel。
- **被谁依赖**：T13 canvas（文档引用校验）、T18 app（资源删除前引用检查）。
- **边界与观察**：引用解析只认文档 JSON 内的规范化路径，`isCanvasCollectionPath`/`isBareResourceIDField`/`isResourceLocatorField` 三个谓词决定字段是否参与引用图；改画布 JSON 结构时必须同步这三个谓词。

## T10 auth

- **职责**：账号、会话、OAuth 登录族。
- **关键符号**：`backend/internal/auth/auth.go` 的 `SessionCookieName="open_ai_canvas_session"`、`sessionMaxAge=30*24h`、`Login`、`Logout`、`CurrentUser`、`createAuthSession`（会话凭证形状 `session.ID + "." + token`）、`HashToken`；OAuth 族文件：`backend/internal/auth/oidc.go`、`backend/internal/auth/wechat.go`、`backend/internal/auth/linuxdo.go`、`backend/internal/auth/libtv.go`+`libtv_oauth.go`、`backend/internal/auth/tapnow.go`、`backend/internal/auth/turnstile.go`。
- **依赖谁**：T04、T01、`internal/sms`（经 `SetSMSDelivery` 注入，包本身不 import sms）。
- **被谁依赖**：T18 app（`authHost` 桥接）、T19 handler。
- **边界与观察**：全包无 JWT——token 明文段只在 Cookie 内存续，库内仅存 `HashToken` 摘要；会话过期即 `sessionMaxAge` 30 天，无滑动续期。

## T11 skills

- **职责**：技能包安装（含 GitHub 拉取）、访问授权、策展。
- **关键符号**：`backend/internal/skills/service.go` 的 `New(repo, dataDir, runWorkerLoop)`；`backend/internal/skills/authorization.go` 的 `IsAdministrator`/`RequireSkillAccess`/`RequireAdministrator`/`requireSkillAdmin`；`skill_packages.go` + `skill_packages_archive.go` + `skill_packages_edit.go` + `skill_packages_files.go` + `backend/internal/skills/skill_packages_github.go`（仅接受公开 `https://github.com` 仓库地址，拒绝带用户信息 URL；经 `api.github.com` 解析 commit，再从 `codeload.github.com` 拉 zip 解包）；`skills_admin.go`、`skills_builtin.go`、`skill_curation.go`、`skill_curation_roots.go`、`seed/`。
- **依赖谁**：T04、T01。
- **被谁依赖**：T18 app（`Service.skills` + `startSkillSyncWorker`/`startEcommerceSkillSyncWorker` 两个同步 worker）。
- **边界与观察**：GitHub 拉取是出站白名单域（github.com/api.github.com/codeload.github.com 三域名硬编码），不经过 T05 的通用白名单机制；`runWorkerLoop` 由 app 层注入，包内不自起 goroutine。

## T12 prompts

- **职责**：提示词模板、Agent 系统策略。
- **关键符号**：`backend/internal/prompts/prompt_template.go` 的 `promptPlaceholderPattern`（`{{...}}` 占位符正则）、`PromptOperationDefinition`/`PromptTemplateRequest`/`CompiledPrompt`、`EnsureDefaultPromptTemplates`、`PromptDefinition`；`backend/internal/prompts/agent_policy.go` 的 `//go:embed agent-system-policy.md`（策略正文内嵌二进制）、`LoadAgentPolicies()`（返回 system/media 两份 `Policy`）、`hashPolicy`（按 id+version+正文哈希）、`parsePolicyDocument`；配套 `prompt_template_defaults.go`、`previs_manual.go`（embed 预演手册）、`storyboard_default.go`、`style_profile.go`。
- **依赖谁**：T04、T03。
- **被谁依赖**：T18 app（`Service.prompts`，经 `promptAdminGate` 桥接管理员判定）。
- **边界与观察**：策略正文随二进制发布（embed），运行期只读；模板占位符未闭合会直接编译失败而非静默替换。

## T13 canvas

- **职责**：画布文档域 + 实时协作 hub。
- **关键符号**：`backend/internal/canvas/realtime.go` 的 `RealtimeHub`（进程内 `rooms` 房间表 + Redis 发布，`NewRealtimeHub(client)`、`Open`/`Changed`/`deliver`/`publish`）；`backend/internal/canvas/realtime_devices.go` 的 `claimRealtimeDevice`、`renewRealtimeDevice`（两个 `redis.NewScript` Lua 脚本做设备槽认领/续约）；`edit_leases.go`（编辑租约）；`backend/internal/canvas/document/canonical.go`（文档规范化）；其余 `operations.go`、`canvas_history.go`、`canvas_share.go`、`repair_*.go`。
- **依赖谁**：T04、T01、go-redis。
- **被谁依赖**：T18 app（`newService` 第一步就建 hub 并用 `repo.WithCanvasChangeNotifier(canvasRealtime.Changed)` 把仓库写路径反接进实时广播）、T19 handler（`RegisterCanvasRealtimeRoutes`）。
- **边界与观察**：设备槽与限流是两套独立 Lua 脚本（本包与 T06 各管各的键空间）；`Changed` 是仓库→hub 的唯一回调入口，漏包一层 notifier 实时性即失效。

## T14 storage

- **职责**：对象存储签名与多厂商适配。
- **关键符号**：`backend/internal/storage/objects.go` 的 `SignedOSSObjectURL`/`SignedOriginObjectURL`（HMAC-SHA1，密钥 `setting.AccessKeySecret`）、`PutOSSObjectContext`（按 `setting.Provider` 分派：腾讯 COS、七牛、火山 TOS、S3、默认阿里云 OSS）；`backend/internal/storage/settings.go` 的 provider 常量实测五种：`aliyun`（默认）、`tencent`、`qiniu`、`volcengine`、`s3`；COS 端点可由 region 推导（`https://cos.<region>.myqcloud.com`）；`Runtime.ProviderAccessURLTTL` 默认 4 小时；`backend/internal/storage/s3.go`（S3 与 TOS 共用签名路径）、`volcengine.go`、`delivery.go`。
- **依赖谁**：T05 outbound（对象下载 URL 也要过出站校验）、aws-sdk 签名器、qiniu/cos SDK。
- **被谁依赖**：T18 app 的 `storage_adapters.go`（`ossSettingValue`/`ossObjectStream` 类型别名层）与 `storage_location.go`（`TestAdminOSSSetting`/`verifyOSSConnection` 连通性测试、`upsertStorageLocation`）。
- **边界与观察**：本包 provider 枚举无「local」——本地文件系统兜底发生在 app 层 `dataDir`（如 `storage_migration.go` 的 `backups` 目录），不是 storage 包的一种 Provider；对象写入整链受 ctx 约束（含 SDK 内部重试），这是注释明示的契约。

## T15 payment

- **职责**：支付插件注册表与进程 ABI 宿主侧。
- **关键符号**：`backend/internal/payment/registry.go` 的 `Registry`、`NewRegistry(providers...)`、`Get`/`Descriptors`/`Providers`；`backend/internal/payment/rpc.go` 的 `pluginRPCVersion="yingce.payment/v1"`、`RPCProvider`、`NewRPCProvider(descriptor, packageDir, entry)`、`ResolveRPCBackendPath`、`classifyPaymentPluginStartError`、`validatePaymentExecutablePlatform`。
- **依赖谁**：os/exec（子进程 ABI）、T01 错误类型。
- **被谁依赖**：T18 app（`Service.paymentRegistry`，被 `pluginRuntime.paymentRegistrySnapshot()` 动态覆盖）。
- **边界与观察**：越界事实——`backend/payment-plugins/`（alipay/epay/wechat/xunhupay/zhifufm 插件实现 + `rpc_runner.go` + `crypto.go`，共 15 个 go 文件）与 `backend/payment-sdk/types.go` 位于 backend 根目录而非 `internal/`，是「可独立分发的插件侧代码」与「不可导出的宿主内部」的物理分界；改 `yingce.payment/v1` 消息形状必须同时动这两处。

## T16 agent

- **职责**：Pi Agent 运行时的两种宿主形态：内嵌子进程与远程 HTTP。
- **关键符号**：`backend/internal/agent/runtime/runtime.go`——内嵌路径：`StartBridge("127.0.0.1:0", "127.0.0.1", bridge)`（回环 Bridge）+ `exec.CommandContext(ctx, "node", "--max-old-space-size="+nodeMemoryMB(), <runtimeDir>/agent-runtime.mjs)`；runtime 目录探测顺序 `CANVAS_PI_RUNTIME_DIR` → `/app/backend/agent-runtime/pi`；进程闸门环境变量 `YINGCE_AGENT_SESSION_CAP`、`CANVAS_AGENT_MAX_PROCESSES`、`YINGCE_AGENT_QUEUE_TIMEOUT_SECONDS`（60–600 秒有界）、`YINGCE_AGENT_MAX_QUEUE`（64–1024）、`YINGCE_AGENT_NODE_MEMORY_MB`。`backend/internal/agent/yingceagent/client.go`——远程路径：`GET <endpoint>/healthz` 探活、`PUT <endpoint>/v1/limit` 限流同步、`POST <endpoint>/v1/runs`（`Authorization: Bearer <serviceToken>`）提运行；`StartBridge("0.0.0.0:0", bridgeHost, bridge)` 绑全部网卡供远程 Node 回连；文件头注释明示「remote endpoint never falls back to an embedded Node process」——远程不可用绝不回退内嵌。
- **依赖谁**：os/exec、net/http、T01。
- **被谁依赖**：T18 app（`cloud_agent_pi_coordinator.go` 的 `startCloudAgentPi`/`resumeCloudAgentPi`/`recoverCloudAgentPiRunners` 经它拉起/恢复会话）。
- **边界与观察**：两种形态安全性不同——内嵌 Bridge 只听回环，远程 Bridge 听 `0.0.0.0`（依赖网络层隔离）； Bridge 回调与 agent 会话是两跳：Node 进程持 `bridgeURL`+`bridgeToken` 回调宿主取模型凭证与工具执行。

## T17 集成域

- **职责**：外围渠道与内容域聚合（短信、邮件身份、MCP、飞书、工具、工作台、电商、剪映草稿）。
- **关键符号**：`backend/internal/sms/service.go`；`backend/internal/emailidentity/email.go`；`backend/internal/integrations/feishu/`（`callback.go`/`client.go`/`digest.go`）；`backend/internal/integrations/mcp/`（`access.go`/`admission.go`/`client.go`/`config.go`/`execution.go`/`sanitize.go`/`service.go`）；`backend/internal/tools/tools_legacy.go`——改号迁移事实：v1.7.3.4/v1.7.3.5 曾把故事板放编号 90/91，与用户自建工具自增编号撞号，现迁至 1000090/1000091，旧内置行停用、收藏迁新编号、旧提示词标签按新编号解析（`LegacyBuiltinTool` + `legacyBuiltinTools` 表）；`backend/internal/workbench/`（`catalog.go`/`seed.go`/`service.go`/`tasks.go`）；电商四包 `ecommerceagent/`（30+ 知识与编排文件：`platform_knowledge.json`、`orchestration_simulation.go`、`shot_assignments.go` 等）、`ecommercecatalog/catalog.go`、`ecommerceknowledge/`、`ecommerceskills/`；`backend/internal/jianying/draft.go`（剪映草稿）。
- **依赖谁**：各自依赖 T04/T01；`tools` 经 `tools_legacy.go` 兼容历史数据。
- **被谁依赖**：T18 app（sms 经 `SetSMSDelivery` 注入 auth；workbench/prompts 经 adminGate 桥接；电商域由专门 worker 同步）。
- **边界与观察**：`integrations/mcp` 的私网放行走 T05 的 `CANVAS_MCP_ALLOWED_PRIVATE_HOSTS`，`sanitize.go` 是 MCP 入参净化闸；tools 改号是唯一显式数据迁移式兼容层，新工具编号必须避开 90/91 与 1000090/1000091。

### 其余域包（主表 T17 全集补齐）

| 包 | 规模 | 职责与关键符号 | 边界/审计要点 |
| --- | --- | --- | --- |
| `internal/generation` | 6 文件 668 行 | 声明式协议执行引擎：`Engine` 经 `Deps` 注入媒体解析/进度/计费钩子/渠道限流/持久化五接口，挂接点 `app/generation_registry_bridge.go`；`AgentProtocolAdapterForContext`、`CanonicalAgentRequest` | `registry.go` 的 `loadOfficialFallbackRegistry` 与 `types.go` 部分符号生产不可达（见 05 账本四） |
| `internal/imagebilling` | 2 文件 203 行 | 图像产出像素计价：`pricing.go` 的 `Amount`/`MapPrices`/`Reserve` | 计价规则独立成包，经 `app/credit_cost.go` 进入积分结算 |
| `internal/hostupdate` | 13 文件 2,808 行 | Host Updater 域：`Manager`（`NewManager`/`StartUpdate`/`runUpdate` 检查-备份-迁移-切换-回滚状态机）、`server.go` Unix socket 服务端、`sources.go` 发布源 | **唯一生产入口是 `backend/cmd/host-updater`**——死代码分析必须覆盖该入口，仅扫 `cmd/server` 会误报 76 个函数（05 账本四的方法学修正） |
| `internal/updaterclient` | 2 文件 154 行 | backend 侧更新客户端：`Client`（`Status`/`Check`/`Start`） | 管理后台系统更新页背后即这条链（与 01 数据流互见） |
| `internal/agentcontext` | 2 文件 226 行 | Agent 检查点契约：`Checkpoint`/`Frame`/`ParseFrame`/`BuildPrompt` | Go 与 Node 两侧共享的契约包 |
| `internal/production` | 14 文件 3,517 行 | 短剧制作域：剧本圣经 `Bible`/`BibleCharacter`、`AssetScheme`/`AssetSettings` | 域内最大包之一 |
| `internal/inspirations` | 4 文件 986 行 | 灵感流：`Feed`/`FeedEntry`、`AdminGate`/`Locker` | worker 由 T18 `StartWorker` 拉起 |
| `internal/poselibrary` | 6 文件 1,541 行 | 姿势库：`importer.go` `Import`、`AdminItem`/`BulkRequest` 管理面 | `Import` 被 `app/pose_library_bridge.go` 调用 |
| `internal/providercatalog` | 6 文件 1,002 行 | 供应商目录运营数据源：`Catalog`/`ApplyMarkup`/`Connection` | T08 provider 经 `providercatalog.Source` 装配插件目录 |
| `internal/buildinfo` | 2 文件 66 行 | 构建信息注入：`Current`/`Info` | 值由构建期注入，运行期只读 |
| `internal/logging` | 4 文件 512 行 | HTTP 访问日志：`http.go` `AccessLog`/`Recovery`、`ConfigFromEnv` | 挂载顺序见 01 接口清单的中间件链 |

## T18 app

- **职责**：全部业务用例的宿主；`Service` 是唯一门面。
- **关键符号与实测规模**：`backend/internal/app` 实测 815 个 go 文件（全仓最大包）；`backend/internal/app/service.go` 的 `Service` 结构体实测 95 个字段（互斥锁族、五个任务协调器、三个 `BoundedReadCache`、piRunner 表、审批收尾表、路由目录快照等）；装配收敛见文首「通用装配顺序」。`StartWorker` 拉起约 12 个后台 worker：`taskWorker`、资源删除/卸载、技能同步、电商技能同步、支付、`startBillingAutoRefundWorker`、Ark 人像、短剧调度、灵感流、消费报告、飞书反馈，外加每 3 秒一次的 `recoverCloudAgentPiRunners` 自旋与每小时一次的瞬态清理。
- **provider 分工**：`backend/internal/app/provider_protocol.go` 头注释「声明式协议插件宿主与接口类型校验」——JSON manifest 插件的宿主运行时（`runDeclarativeProtocolTask`：manifest 只描述请求/响应映射，凭证、出站安全、轮询、结果下载归宿主）；`backend/internal/app/provider_video.go` 头注释「视频生成遗留手写路径；已有官方插件的接口类型走声明式协议」；`backend/internal/app/provider_jimeng.go`（volc-sdk 依赖的即梦手写适配，未并入声明式宿主）。
- **双轨运行时**：`Service.legacyCloudAgentRootTask` 字段——注释明示仅测试开启，走 pre-Pi 模型 worker 路径，运行时可用性不得改变根任务语义；生产路径是 `cloud_agent_pi_*.go` 文件族（`cloud_agent_pi_bridge.go`/`cloud_agent_pi_coordinator.go`/`cloud_agent_pi_events.go`/`cloud_agent_pi_model.go`/`cloud_agent_pi_helpers.go`）。
- **配额与退款**：`backend/internal/app/storage_quota.go` 的配额链：结构化容量（`structuredBytes = AssetBytes + CanvasBytes`）、任务条数、任务数据增长、上游请求日志条数四级校验函数，其中任务条数与日志条数两支共用 `validateTaskDataGrowthQuotaWithPolicy` 的数据量增长闸——两跳闸门串联，非三段独立计费；`backend/internal/app/billing_auto_refund.go` 的 `BillingAutoRefundSetting{Enabled}` 默认关闭（头注释明示），开启走管理审计（`billing_auto_refund.update`），关闭时 worker 直接跳过。
- **依赖谁**：几乎全部内部包（import 实测含 repository/auth/canvas/inspirations/kernel/model/payment/platform/poselibrary/prompts/providercatalog/skills/sms/tools/workbench）。
- **被谁依赖**：仅 T19 handler 与 T21 cmd。
- **边界与观察**：`taskWorkerConcurrency=3`、`taskLogPayloadLimit=4000` 为包级常量；`dramaTaskHook` 等测试钩子字段在生产恒空，读代码时勿把钩子路径当生产路径。

## T19 handler

- **职责**：Gin 路由面：全部 `Register*` 入口与流式端点。
- **关键符号**：`backend/internal/handler/api.go` 的 `RegisterCanvasAPI`——实测 53 处 `Register*` 调用（含 `RegisterOpenAPIRoutes` 暴露 `/openapi.yaml`），第一行即挂 `StorageReadOnlyGuard` 中间件（注释：必须先于所有业务路由，Gin 中间件只作用于其后注册的路由）；短剧域路由单独分组挂 `RequireFeature(svc, service.FeatureShortDrama)`。`backend/internal/handler/routes.go` 的 `RegisterTaskRoutes` + `streamTaskTextEvents`（任务文本增量回放轮询实测 750ms tick）。`backend/internal/handler/auth.go` 的 `RegisterAuthRoutes`/`RegisterOAuthCallbackRoutes`/`RegisterAdminRoutes`/`RegisterSystemProxyRoutes`/`SystemProxyNoRouteHandler`（短式公开代理路径的 NoRoute 兜底，内部经 `SystemProxyContract.DeclaredPostPaths()` 做方法+路径白名单）。`backend/internal/handler/custom_proxy.go` 的 `RegisterCustomRelayRoutes`；`backend/internal/handler/canvas_realtime.go` 的 `RegisterCanvasRealtimeRoutes`；`backend/internal/handler/system_proxy_stream.go` 的 `isSystemProxyEventStream`/`copySystemProxyResponseHeaders`/`streamSystemProxyResponse`（SSE 透传与响应头复制）。
- **依赖谁**：T18（`*service.Service` 单参数注入全部业务）、gin。
- **被谁依赖**：T21 `cmd/server` 把 `RegisterCanvasAPI` 挂在 `/api`。
- **边界与观察**：包内 `func Register` 定义实测 57 个；新路由不进 `RegisterCanvasAPI` 即不可达——这是「加接口必须改两处」的硬检查点；版本策略是无前缀（注释：前后端一起发布，不做路径版本前缀）。

## T20 service

- **职责**：handler→app 的再导出壳，纯转发。
- **关键符号**：实测 24 个 `backend/internal/service/aliases_*.go`（`aliases_admin_assistant.go`、`aliases_agent_memory.go`、`aliases_browser_agent.go`、`aliases_canvas_realtime.go`、`aliases_channel_websites.go`、`aliases_consts.go`、`aliases_creation_history.go`、`aliases_drama.go`、`aliases_feedback.go`、`aliases_feishu.go`、`aliases_funcs.go`、`aliases_inspiration_feed.go`、`aliases_login_logs.go`、`aliases_mcp.go`、`aliases_notifications.go`、`aliases_official_assets.go`、`aliases_pose_library.go`、`aliases_profile.go`、`aliases_referral.go`、`aliases_site_traffic.go`、`aliases_skill_curation.go`、`aliases_team_assets.go`、`aliases_types.go`、`aliases_wechat.go`）+ `backend/internal/service/doc.go`。
- **依赖谁**：T18。
- **被谁依赖**：T19（handler 只 import service，不直接 import app）。
- **边界与观察**：任何真实逻辑进这个包都是违例；handler 签名里的 `*service.Service` 实为 `app.Service` 的别名，改名要同时动 T18/T19/T20 三层。

## T21 cmd

- **职责**：进程入口集合。
- **关键符号**：实测 `backend/cmd/` 下 14 个入口：`server`（`main.go` + `system_status.go`）、`migrate-schema`、`migrate-sqlite-postgres`、支付插件五个 `payment-alipay`/`payment-epay`/`payment-wechat`/`payment-xunhupay`/`payment-zhifufm`、`host-updater`；另有五个数据工具入口：`admin-assistant-catalog`、`ecommerce-knowledge`、`migrate-channel-model-price-tiers`、`migrate-logical-model-families`、`reseed-logical-model-sources`。
- **依赖谁**：T18/T19/T02。
- **被谁依赖**：部署层（`backend/Dockerfile` 与根 `Dockerfile` 的构建产物、`scripts/install-server-image.sh`）。
- **边界与观察**：支付五入口与 T15 的 `backend/payment-plugins/` 一一对应——每个入口包一个插件的可执行宿主；`migrate-channel-model-price-tiers`/`migrate-logical-model-families`/`reseed-logical-model-sources` 是与 T02 版本迁移正交的数据订正工具，跑一次即弃，不进服务启动链。
