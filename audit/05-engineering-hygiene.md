# 05 · 工程卫生审计（量化现状）

**一句话结论**：仓库活代码纪律良好（TODO 仅 2 处、活代码重复仅 570 行），债务集中在四类「历史地层」——前端死文件 45 个 35,408 行（含编号快照 CSS 11 个文件 30,282 行零引用）、Go 不可达函数 275 个（含手写内置协议目录整包 1,275 行）、git 内约 190MB 编译产物（pack 544MiB）、services→stores 反向依赖 64 处；同批债务在原始仓 `ddcat-ai/open-ai-canvas` main 上逐字节存在。

## 体量分布

| 桶 | 代码行 | 文件数 |
| --- | ---: | ---: |
| src | 648,714 | 2,854 |
| tests | 195,037 | 1,384 |
| docs & assets | 331,896 | 1,699 |
| scripts | 9,172 | 28 |

三个规模事实：`web/src/styles` 仅 23 个文件承载 63,436 行 CSS；`backend/internal/app` 单包 115,721 行非测试源码（815 个 `.go` 文件，与 04 的 T18 卡一致）；TypeScript 353,807 行为第一大语言。

> 下钻：[债务台账明细](details/05-debt-ledger.md) —— 全量账本、甄别口径、上游对比与复现命令。

## 死代码

- **前端（knip）**：`src/` 死文件 45 个 / 35,408 行，按域集中在旧一代电商页面（`pages/ecommerce` 13 个）、编号快照 CSS（`styles` 11 个）、画布旧组件与 lib（9 个）；抽样经 codegraph `callers` 与 tsserver `find_usages` 双验证零引用。knip 另报的 `test/` 70 个为 bun test 约定发现的误报，不计入。
- **编号快照 CSS**：`globals-1..7.css` + `admin-ui-1..4.css` 共 30,282 行，全仓零 `import`，jscpd 证实与现行 `globals.css`/`admin-ui.css` 克隆 17,080 行——是旧版式的冻结快照而非复制债。
- **后端（deadcode，全 14 个 cmd 入口）**：275 个不可达函数 = 210 个完全无引用 + 65 个仅测试引用；最大集中地是手写内置协议目录 `internal/protocol/builtin.go`（55 个，整包 1,275 行生产不可达，调用链闭合验证至 `app/provider_protocol.go` `validateGenerationInterface` 与 `internal/generation/registry.go` `loadOfficialFallbackRegistry`）。方法学要点：仅扫 `cmd/server` 会把 hostupdate 域 76 个函数误判为死码，必须全入口扫描。

## 重复代码

jscpd 538 个克隆 / 17,893 行，约 95% 来自编号快照 CSS（17,080 行）；活代码间 TS 重复仅 570 行，已知复制家族为 `depth/lineart/pose-runtime.ts` 三胞胎等 5 组小簇（明细见台账五）。

## 仓库卫生

`backend/server`（约 62.7MB，blob `cfba15a`）与 `plugin-packages/` 35 个多平台 provider 二进制入库，git pack 达 544.09 MiB；`.scratch/` 8 个工作笔记被跟踪；`backend/agent-runtime/pi/package-lock.json` 与根 `AGENTS.md` 锁文件条文字面冲突；根 `CONTEXT.md` 自 2026-09 未更新、职能与 `audit/` 重叠。依赖面：`mammoth`、`@streamdown/code` 已装未用，`shadcn`（CLI）位列运行时依赖且零引用，`shiki` 被引用却未声明（明细见台账二）。

## 上游基线

原始仓 main（`a8fca0fa`）与本仓分叉 45/11 提交（本地领先/落后）；同批债务在上游逐字节存在（含同一 blob），Go 死函数上游 289 个——本地修活 43、新引入 29，净减 14。结论：拔债补丁两侧通用。

## 反向依赖

services→stores 运行时 import 64 处（`services/api` 内 30 处）；与 04 的文件口径（51 文件、32 个依赖 `stores/use-config-store`）兼容，本篇为 import 行口径。

## 审计观察（现状事实）

1. **快照地层未拔**：11 个编号 CSS 文件 30,282 行零引用，仍留在源码树（`web/src/styles/`）。
2. **整包旧产品地层**：旧一代电商页面族 13 个文件在 `src/pages/ecommerce` 内部自洽、外部零进入（`use-result-node-drag.ts` 仅被同样死亡的 `result-canvas.tsx` 引用）。
3. **协议双轨的死侧已失守**：手写内置协议目录生产不可达，仅 "legacy tests and migration tools" 与其自身测试引用（`backend/internal/protocol/builtin.go`、`app/provider_protocol.go` 注释自述）；`provider_video.go` 的遗留渠道兼容分支与它不是同一层——前者仍可达。
4. **OSS 适配未接线**：`internal/storage/objects.go` `PutAliyunOSSObject` 零调用方。
5. **二进制出库缺位**：`backend/server` 与插件 provider 二进制入库且上游同源，pack 544MiB。
6. **多入口扫描缺位**：仓库无任何死代码门禁；`scripts/verify-before-push.mjs` 不含 knip/deadcode 基线检查。
7. **反向依赖族最重点位**：`services/user-data-sync.ts` 单文件 6 条 store import（与 04 一致）；api 层 30 处反向依赖中含运行时 import（如 `services/api/generation-task.ts` 引 `stores/use-config-store`）。
