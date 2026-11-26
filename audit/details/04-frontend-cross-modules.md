# 前端与横切模块细节卡（T22–T32）

本文件是 [04 技术模块审计](../04-tech-modules.md) 编号表的细节增量：卡片只补符号、规模与边界事实，全部数字实测于当前代码树。

## T22 lib

- **职责**：纯函数、协议转换与画布算法层。
- **关键符号与规模**：实测 117 项（14 个子目录）；`lib/canvas/` 250 文件（node-registry/、tool-registry/、previs/、image-mask-document、canvas-content）；`lib/model-protocols/`、`lib/gemini-image`、`lib/art-critique/`（pipeline、review）、`lib/plugins/`（plugin-types、plugin-registry、plugin-storage、builtin/、official-applications、plugin-search）、`lib/timeline/`、`lib/drama/`、`lib/workspace-route-modules.ts`、`lib/preloadable-lazy.ts`。
- **依赖谁**：原则上仅标准库与第三方纯库。
- **被谁依赖**：services、stores、components、pages 全层。
- **边界与观察**：两处运行时反向依赖 services（见文末耦合清单第 8、9 条）；`lib/plugins/plugin-types.ts` 的 `PluginManifest` 与后端 `PluginManifestView` 是同一清单的前后端双视图。

## T23 services/api

- **职责**：唯一 HTTP 边界与全部协议组包。
- **关键符号与规模**：实测 93 文件；`request.ts` 的 `http`（信封 `BackendEnvelope{code,data,msg,reason}`、`ApiError` 含业务码与 304/Retry-After 语义）；渠道传输 `custom-channel-relay.ts`（channelRequest 改写 `/ai/custom`）与 `channel-transport.ts`（createChannelTransport：image/video/audio 场景，附 X-Canvas-Scene 与 X-Idempotency-Key）；图像 `image.ts` 拆 image-transport/image-protocols/image-streaming/image-response/image-validation；视频 `video.ts` 按 provider 分派（video-provider-* 族 + video-transport）；音频 `audio.ts`（minimax-speech、evolink-seed/qwen-audio、async-audio、OpenAI speech）；SSE 解析 `task-text-stream.ts`。
- **依赖谁**：axios、T22、（反向）T25 use-config-store。
- **被谁依赖**：T24、stores、components、pages。
- **边界与观察**：Provider 特有 payload/解包留在各自模块，不进通用 request；任务 SSE 游标语义见 03-module-flows。

## T24 services

- **职责**：跨页面副作用（缓存、同步、生成消费、Agent 运行时接入）。
- **关键符号与规模**：实测约 90 文件；媒体缓存族 image-storage/file-storage/resource-blob-cache/local-media-object-urls；同步族 canvas-realtime(-client)、canvas-sync-polling、canvas-mutation-batches、canvas-edit-leases、user-data-sync(-media)、project-asset-sync、creation-history-sync；生成副作用族 generation-artifact-sink/generation-asset-repository/generation-completion-tracker/generation-consumer-dedupe/generation-consumer-lifecycle/generation-task-materializer/canvas-generation-consumer/provider-neutral-generation-effects；Agent 族 `services/agent/`（browser-agent worker/protocol/host/loop）+ creative-agent-controller + cloud-agent-*；`local-runtime-session.ts`。
- **依赖谁**：T23、T25、T22。
- **被谁依赖**：stores、components、pages。
- **边界与观察**：与 stores 双向耦合最重的一层（见文末耦合清单第 1–4 条）；画布同步三轨并存（实时/轮询/写侧协调）。

## T25 stores

- **职责**：跨页面状态与持久化配置。
- **关键符号与规模**：根约 20 个 `use-*-store`（user/auth/config/model-catalog/asset/plugin/appearance/theme/locale/sync-progress/global-agent 等）+ `stores/canvas/` 6 个（use-canvas-store 872 行 persist、history、ui、theme、prompt-preferences、previs-workbench）。
- **依赖谁**：zustand、T24/T22（use-asset-store import image-storage/file-storage/generation-asset-repository）。
- **被谁依赖**：components、pages、（反向）services/api。
- **边界与观察**：媒体大对象不进 localStorage；use-config-store 是被 services 反向依赖的焦点（耦合清单第 1–4 条）。

## T26 components

- **职责**：跨页复用 UI 与交互能力。
- **关键符号与规模**：实测 52 项；`components/canvas/` 232 文件为最大子域（canvas-viewport、canvas-node-*、canvas-workspace-*、toolbars/、nodes/、previs/）；跨页件 model-picker、image-settings-panel、audio-settings-panel、media-preview、model-editor-modal、auth/、layout/、ui/。
- **依赖谁**：T25、T24、T23、T22（反向依赖 pages 的 3 处见耦合清单）。
- **被谁依赖**：pages、layouts。
- **边界与观察**：canvas 子域与 stores/canvas、lib/canvas 构成画布三件套（视图 232 / 状态 6 store / 逻辑 250 文件）。

## T27 layouts 与路由装配

- **职责**：路由级布局、全局浮层与应用装配。
- **关键符号**：`web/src/layouts/user-layout`；`web/src/router.tsx`（`preloadableLazy`/`deferred`/`fullScreenDeferred` 三种懒加载包装、RequireAuth/RequireFeature/RequireSkillAdmin 守卫、admin 段经 `pages/admin/admin-page-loader` 的 `lazyAdminPage`）；`web/src/application.tsx`。
- **依赖谁**：T26、T28、T25。
- **被谁依赖**：应用入口。
- **边界与观察**：两套懒加载机制并行（router 侧与 admin 侧），语义不同（预加载包装 vs 独立 loader）。

## T28 pages

- **职责**：路由页面与页面私有 hook/组件。
- **关键符号与规模**：路由映射与守卫全表见 [03-route-inventory](03-route-inventory.md)；`pages/admin/` 独立域（约 40+ 子路由，经 admin-page-loader 懒加载）；`pages/canvas/` 58 文件 hooks 族；`pages/ecommerce/` 约 60 文件；`pages/projects/` + `studio/` 约 35 文件。
- **依赖谁**：全层。
- **被谁依赖**：仅路由装配。
- **边界与观察**：DEV 实验页（pages/dev/、test-voice-recording）源码常驻产品目录，构建期摇树。

## T29 agent-runtime/pi（横切）

- **职责**：Node 单次运行 CLI：stdin 读 JSON 请求，驱动 pi SDK 执行一轮 Agent 步骤，stdout 逐行输出事件。
- **关键符号与规模**：`backend/agent-runtime/pi/agent-runtime.mjs`（bridge() 回调 Go 侧 Bridge）；依赖 pi-coding-agent + pi-ai（1.5.2）、typebox、undici；Node ≥ 22.19。
- **边界与观察**：SDK 模型/工具调用不直连上游，全部经 Bridge 回调宿主取凭证与工具执行。

## T30 yingce-agent 容器（横切）

- **职责**：远程 Agent 执行服务。
- **关键符号**：`yingce-agent/server.mjs`（bearer 常时比较鉴权、SessionAdmission 并发闸门：MAX_CONCURRENT_SESSIONS 默认 8 硬顶 64、队列与超时、RunDirectories 孤儿目录回收）；健康检查 `/healthz` 8081；每请求再 spawn 一个 agent-runtime.mjs。
- **边界与观察**：与内嵌模式由 app 层单点互斥（cloud_agent_pi_runtime.go）；只读加固 + tmpfs + cap_drop 部署。

## T31 部署编排（横切）

- **职责**：构建与运行拓扑。
- **关键符号与规模**：根 Dockerfile（bun+node 构建 Vite，构建前跑 module-preload 测试；nginx 1.27 运行镜像，dist 入 /opt/canvas-release，entrypoint docker/canvas-web-entrypoint.sh）、backend/Dockerfile（支付插件包构建冒烟 → 三个 Go 二进制 → pi node_modules → alpine+ffmpeg 非 root）、yingce-agent/Dockerfile；`nginx.conf`（SPA 回退、/assets 哈希长缓存、/api 反代 3600s、SSE 正则 location 关缓冲、WebSocket Upgrade）；7 份 compose（dev/local/server/deploy/browser-only/build/根）。

## T32 工具链（横切）

- **职责**：安装、验证与本地启动脚本。
- **关键符号**：`scripts/install-server-image.sh`（GHCR Release 镜像轨 + Host Updater）、`scripts/install-server.sh`（源码构建轨，deploy+build 叠加）、`scripts/install-host-updater.sh`（digest 固定强校验）、`scripts/verify-before-push.mjs`（口径对齐 CI）、`scripts/check-docs-index.mjs`（docs+audit 两域自检）、`scripts/start-local.ps1`、`start-canvas.ps1`、`start-whisper-local.sh`、`windows-proxy.ps1`、`audit-source-overlap.py`。
- **边界与观察**：`launch-canvas.ps1` 硬编码本机绝对路径与隧道域名（机器绑定脚本入仓，见 01 审计观察）。

## 反向耦合证据清单（依赖方向核查）

| # | 方向 | 证据（路径 → 被依赖符号） |
| --- | --- | --- |
| 1 | services → stores | `web/src/services/api/image.ts` → `stores/use-config-store`（resolveModelRequestConfig） |
| 2 | services → stores | `web/src/services/api/video.ts` → use-config-store |
| 3 | services → stores | `web/src/services/api/audio.ts` → use-config-store |
| 4 | services → stores | `web/src/services/api/channel-transport.ts` → use-config-store（AiConfig） |
| 5 | stores → services | `web/src/stores/use-asset-store.ts` → services 的 image-storage/file-storage/generation-asset-repository |
| 6 | components → pages | `web/src/components/canvas/canvas-cloud-agent-settings.tsx` → `pages/settings/` agent-memory-pane |
| 7 | components → pages | `web/src/components/canvas/canvas-workspace-panel.tsx` → `pages/canvas/` use-canvas-workspace-tasks |
| 8 | components → pages | `web/src/components/canvas/canvas-workspace-personal-asset-preview.tsx` → `pages/assets/` asset-library-panels |
| 9 | lib → services | `web/src/lib/art-critique/pipeline.ts` 与 `review.ts` → `services/api/image`（requestToolResponse） |
| 10 | lib → services | `web/src/lib/canvas/agent-error-presentation.ts` → ApiError |

> 主方向 `pages → components → stores → services → lib` 成立；上表是全部已实证的反向边。其中第 1–4 条（services 反向依赖 use-config-store）为耦合最重的族。
