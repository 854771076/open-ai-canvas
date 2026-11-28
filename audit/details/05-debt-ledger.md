# 债务台账明细（量化现状）

本页是 [05 工程卫生](../05-engineering-hygiene.md) 的账本明细：全部数字为实测值，附工具、口径与复现命令。**测量基线：main `cdff5336`（2026-11-27）**；代码持续演进，以下数字随 HEAD 漂移，处置前请按文末命令复验。

工具口径：行数统计用 cloc 2.12（物理代码行，剔除空行/注释，仅 git 跟踪文件）；前端死文件用 knip；Go 不可达函数用官方 `golang.org/x/tools/cmd/deadcode`（覆盖 `backend/cmd/` 全部 14 个二进制入口）；重复块用 jscpd（`--min-tokens 150`）；引用复核用 codegraph 符号图（4,004 文件 / 248,101 边）与 VS Code tsserver 交叉验证。

## 一、体量四桶

| 桶 | 代码行 | 文件数 | 主要内容 |
| --- | ---: | ---: | --- |
| src | 648,714 | 2,854 | `web/src`（353,807 行 TS）+ backend Go 源码（1765 文件 335,983 行）+ agent runtime |
| tests | 195,037 | 1,384 | `web/test`（74,369 行）+ Go `*_test.go`（119,392 行） |
| docs & assets | 331,896 | 1,699 | `docs/`、`backend/builtin/`（155,231 行内置技能内容）、`web/public/`、插件包文档 |
| scripts | 9,172 | 28 | 仓库脚本与工具 |

语言头部：TypeScript 353,807 / Go 335,983 / Markdown 212,376 / JSON 130,667 / CSS 122,525。CSS 仅 23 个文件承载 63,436 行（见三）；`backend/internal/app` 单包 115,721 行非测试源码 + 74,690 行测试 + 11,335 行 testdata/JSON。

src 分域排行（前 12，物理代码行）：

| 域 | 代码行 | 文件数 |
| --- | ---: | ---: |
| `backend/internal/app` | 201,746 | 827 |
| `backend/builtin` | 155,231 | 1,138 |
| `web/src/pages` | 138,867 | 478 |
| `web/src/components` | 93,801 | 419 |
| `web/test` | 74,369 | 622 |
| `web/src/styles` | 63,436 | 23 |
| `web/src/lib` | 45,056 | 420 |
| `backend/internal/handler` | 30,132 | 153 |
| `backend/internal/repository` | 29,729 | 178 |
| `web/public` | 29,556 | 73 |
| `web/src/services` | 25,672 | 176 |
| `web/src/i18n` | 24,530 | 61 |

**巨型文件**：TS/Go/CSS 中 ≥1,000 行的文件 71 个，其中 ≥2,000 行 26 个。头部：`web/src/styles/globals.css`（25,181）、`web/src/styles/admin-ui.css`（9,831）、`web/src/pages/projects/drama.css`（7,162）、`canvas-previs-workbench.css`（3,952）、`canvas-cloud-agent.css`（3,773）、`web/src/pages/canvas/project.tsx`（3,718，最大 TSX）、`ecommerce/studio.css`（3,373）、`workspace.css`（3,324）、`backend/internal/app/provider_test.go`（3,059）、`canvas-cloud-agent-panel.tsx`（2,740）。CSS 单体过大是该维度的主要构成。

## 二、前端死文件台账（knip）

knip 报 122 个未引用文件，甄别后分三类：

| 类别 | 数量 | 物理行数 | 判定 |
| --- | ---: | ---: | --- |
| `src/` 死文件 | 45 | 35,408 | **真实死代码主体**。抽样经 codegraph `callers` 与 tsserver `find_usages` 双验证为零引用（如 `CanvasWorkspaceToolPanel`、`CanvasSizePicker`、`CanvasCreativeInteraction`、`CreationInspirationTunnel`） |
| `test/` 未引用测试 | 70 | 21,501 | **误报**：bun test 按文件名约定发现测试，不经 import 链，knip 看不到入口 |
| `public/`、`scripts/` 各别 | 7 | — | **需逐条甄别**：`public/live2d/live2dcubismcore.min.js` 等为运行时按 URL 加载；`scripts/check-changed-formatting.mjs` 被 `scripts/verify-before-push.mjs` 以子进程字符串引用 |

**依赖与导出面（同一次 knip 扫描的其余发现）**：`mammoth`、`@streamdown/code` 已安装未使用；`shadcn`（CLI 工具）位列 `web/package.json` 运行时依赖且源码零引用；`shiki` 被 `web/src/components/ai/ai-message-code-block.tsx` 引用但未在 `web/package.json` 声明（当前靠传递依赖承载）；另有未用导出 637 项与未用导出类型 324 项（上游 638，随死文件拔除自然消减，属增量治理范畴）。

`src/` 45 个死文件按域分布（整代旧地层特征明显）：

| 域 | 文件数 | 说明 |
| --- | ---: | --- |
| `pages/ecommerce` | 13 | 旧一代电商工作台页面（`result-canvas.tsx`、`task-queue.tsx`、`planning-activity.tsx`、`model-picker.tsx`、`generation-confirmation.tsx` 等） |
| `styles` | 11 | 编号快照 CSS，见三 |
| `lib/canvas` | 4 | `cinematic-video-tools.ts`、`canvas-storyboard-operations.ts`、`project-chapter-storyboard.ts`、`agent-canvas-snapshot.ts` |
| `components/canvas`（含 previs） | 5 | `canvas-workspace-tool-panel.tsx`、`canvas-size-picker.tsx`、`previs-viewport-dock.tsx` 等 |
| 其余 9 域 | 12 | `pages/create`（2）、`components/conversation`（2）、`lib`（2）、`services`（2）、`hooks`、`components/layout`、`pages/admin/components` 等 |

死文件族内部自洽：`use-result-node-drag.ts` 仅被同样死亡的 `result-canvas.tsx` 引用——外部零进入的封闭旧地层。

## 三、编号快照 CSS（最大单笔死代码）

`web/src/styles/globals-1.css` … `globals-7.css` 与 `admin-ui-1.css` … `admin-ui-4.css` 共 **11 个文件 30,282 行**：

- 全仓无任何 `import` / `@import` / 动态引用（`web/` 目录含 ts/tsx/css/html 全文检索零命中）。
- jscpd 证实其与现行文件是**冻结快照关系**：与 `globals.css`/`admin-ui.css` 的克隆块合计 17,080 行（如 `globals-1.css` 单文件与现行 `globals.css` 重复 2,218 行）。
- 首部注释自证来源：`globals-1.css` 开头为「Design Token System — 三层架构（Wayfinder #96）」时代的内容，现行 token 体系已在其后另行演进。

## 四、Go 死函数台账（deadcode）

从 `cmd/` 全部二进制入口可达性分析：**275 个不可达函数 = 210 个完全无引用 + 65 个仅测试引用**（测试成了死代码的唯一消费者）。

| 包 | 不可达函数 | 备注 |
| --- | ---: | --- |
| `internal/protocol/builtin.go` | 55 | 手写内置协议目录，见下 |
| `internal/app`（cloud_agent helpers、provider_text_stream、platform/prompts bridge 等） | 约 90 | 含 `PutAliyunOSSObject`（`internal/storage/objects.go`，阿里云 OSS 上传从未接线） |
| `internal/generation`（registry、types） | 19 | `loadOfficialFallbackRegistry` 所在回退链 |
| `internal/protocol/documentation.go`、`builtin_agnes.go` | 18 | 同 builtin 老地层 |
| 其余（hostupdate 之外） | 约 93 | feishu/digest、platform/runtime_policy 等 |

**手写内置协议老地层（整包生产不可达）**：`builtin.go`（858 行）+ `builtin_agnes.go`（304 行）+ `documentation.go`（113 行）≈ **1,275 行**。调用链闭合验证：`protocol.Builtins()` 的全部生产调用点所在函数——`app/provider_protocol.go` 的 `validateGenerationInterface`（包级与 `Service` 方法）与 `internal/generation/registry.go` 的 `loadOfficialFallbackRegistry`——自身全部不可达；`provider_protocol.go` 内注释自述其服务对象为 "legacy tests and migration tools"。`protocol/builtin_test.go`（661 行）在测试这套不可达目录。

**多入口扫描口径**：仅分析 `cmd/server` 时 hostupdate 域曾报 76 个"死函数"；补齐 `cmd/host-updater` 等 14 个入口后该域归零——多入口仓库必须全入口扫描。本轮 275 相对上游 289 的差异见七。

## 五、重复代码台账（jscpd）

538 个克隆 / 17,893 行，归因后**约 95% 来自编号快照 CSS**：

| 归因 | 行数 |
| --- | ---: |
| 编号快照 CSS ↔ 现行 CSS | 17,080 |
| 活代码间 CSS | 243 |
| 活代码 TS/TSX | 570 |

活代码复制家族（量小、可顺手收敛）：`services/depth-runtime.ts` ↔ `lineart-runtime.ts` ↔ `pose-runtime.ts`（同构"本地运行时会话客户端 + store + 限额"骨架，约 62 行 × 3）；`image-parameter-fields.tsx` ↔ `image-settings-panel.tsx`（62 行）；`canvas-choose-effect-picker.tsx` ↔ `canvas-choose-image-style-picker.tsx` ↔ `canvas-choose-motion-picker.tsx`（约 55 行 × 3）；`feedback-modal.tsx` ↔ `feedback-inbox-pane.tsx`；`libtv-import-dialog.tsx` ↔ `tapnow-import-dialog.tsx`。活代码间重复水平整体健康。

## 六、仓库卫生

| 项 | 现状 | 锚点 |
| --- | --- | --- |
| 编译二进制入仓 | `backend/server`（PE 可执行，约 62.7MB，blob `cfba15a`，2026-09-14 入库） | `git ls-files -s backend/server` |
| 插件多平台产物入仓 | `plugin-packages/` 5 个官方支付插件各内嵌多平台 provider 二进制，合计 35 个 `provider*` 文件（单插件包 16–18MB） | `plugin-packages/official-payment-*/backend/` |
| git pack 体积 | 544.09 MiB，二进制是主要成因 | `git count-objects -vH` |
| 工作笔记入仓 | `.scratch/` 8 个 issue 草稿被跟踪 | `.scratch/character-dressing-table/issues/` |
| 锁文件约定冲突 | `backend/agent-runtime/pi/package-lock.json` 入仓，与根 `AGENTS.md`「不提交 package-lock.json」条文字面冲突（该条语境为 web/，但未限定范围） | `backend/agent-runtime/pi/package-lock.json` |
| 根部过时上下文 | 根 `CONTEXT.md`（17 行）自 2026-09-24 未更新，职能与 `audit/` 重叠 | `CONTEXT.md` |
| 忽略规则认知负担 | 根 `.gitignore` 对 `docs/` 采用「整域忽略 + 逐条反向豁免（`!`）」白名单链，新增专题文档必须同步维护豁免条目，漏配即静默不入库 | `.gitignore`（docs/ 段） |

## 七、上游基线对比（origin/main `a8fca0fa`）

原始仓 `ddcat-ai/open-ai-canvas` 与本仓分叉：本地领先 45 个提交、落后 11 个提交（`fork/main` 已完全并入本地）。对同一批债务在上游的实测：

| 债务项 | 上游现状 | 结论 |
| --- | --- | --- |
| 编号快照 CSS / protocol builtin / `backend/server` / 旧电商死页面样本 | 与本地**逐字节相同**（`cmp` 零差异，含同一 blob `cfba15a`） | 债务继承自共同上游，非本地引入 |
| knip src 死文件 | 45 个，抽样全部命中 | 同上 |
| 未用依赖（`mammoth`、`@streamdown/code`、`shiki` 未声明） | 逐项相同 | 同上 |
| Go 死函数 | 289 个 | 本地修活 43 个（如 `advanceCloudAgent` 调度家族）、新引入 29 个，净减 14 |
| services→stores | 62 处（api 层 30） | 与本地 64 处同源 |

## 八、复现命令

```bash
# 体量四桶（仅 git 跟踪文件；输出目录用仓库本地缓存 .local/cache，勿入 git）
git ls-files -z | tr '\0' '\n' > .local/cache/all.txt && bunx cloc --list-file=.local/cache/all.txt --quiet

# 前端死文件
cd web && bunx knip --no-progress --no-exit-code

# Go 不可达函数（必须覆盖全部 cmd 入口）
cd backend && go run golang.org/x/tools/cmd/deadcode@latest ./cmd/...

# 重复块
cd web && bunx jscpd src --min-tokens 150 --reporters json --output ../.local/cache/jscpd

# 反向依赖（import 行口径）
rg -c "from ['\"]@/stores" web/src/services | awk -F: '{s+=$NF} END{print s+0}'
```
