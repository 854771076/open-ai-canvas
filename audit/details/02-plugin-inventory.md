# 插件全量清单：五类插件、包盘点、分发与渠道衔接

本页盘点插件体系的全部类型与实物：五类插件的运行位置、代码入口、清单来源与宿主边界，`plugin-packages/` 的实勘数量分组，官方分发与"插件中心"市场机制，以及与自定义渠道体系的两个衔接点。所有数字与路径均经实地抽查核对。

## 一、五类插件清单

| 类型 | 运行位置 | 代码入口 | 清单来源 | 宿主边界 |
|---|---|---|---|---|
| 协议（请求 provider，declarative） | 后端进程内声明式引擎，无独立进程 | `backend/internal/protocol/manifest.go`（`loadDeclarativeManifest`/`manifestAdapter`）+ `backend/internal/app/protocol_plugins.go`（`pluginRuntime.reload`） | `plugin-packages/*.yingce-plugin`（实勘 113 个 provider 包）或管理员上传件 | 只放行清单声明的相对路径与方法；系统代理按 `SystemProxyContract.DeclaredPostPaths` 白名单；出站鉴权按 `ManifestAuth`；`host:` 执行器仅代码内 bundled 清单可引用，上传件与官方目录包均被拒 |
| 支付（paymentProviders） | 独立 OS 进程：包内二进制 stdin/stdout 单次 JSON RPC | `backend/internal/payment/rpc.go`（`RPCProvider`）+ `backend/cmd/payment-*/main.go`（五入口）+ `backend/internal/app/protocol_plugins.go`（包物化） | `plugin-packages/official-payment-{wechat-native,alipay-page,xunhupay,zhifufm,epay}.yingce-plugin`（5 个，backend=rpc，apiVersion=v1） | 30 秒超时、2MB 输出上限、净化环境变量、可执行文件魔数与架构校验；订单、密钥加密存储、验签判定、入账事务全部留在宿主；无插件时渠道列表为空、下单 fail-closed，绝不回退宿主内置 |
| 短信（smsProviders） | 后端 `host:sms-aliyun` / `host:sms-tencent` 宿主执行器 | `backend/internal/app/sms_plugins.go`（`bundledSMSPluginManifests`）+ `backend/internal/sms/service.go` | 代码内 bundled 清单（`plugin-packages/` 无短信包） | 仅系统来源：上传件或官方目录中的短信贡献一律判 invalid（"短信插件仅支持系统宿主适配器"）；ActivationScope=system，凭据在短信渠道管理中配置 |
| 工作流（workflows） | trusted-backend 适配器（宿主进程内转发） | `backend/internal/app/workflow_plugins.go`（`workflowPluginManifest`、`RequireWorkflowPluginForUser`） | 代码内 bundled 清单 `runninghub-workflow-provider`（backend=trusted-backend）+ `plugin-packages/autodl-comfyui.yingce-plugin`（v2，declarative workflows 贡献） | 接口类型 `runninghub-workflow-image/video/audio` 映射回同一插件 ID；用户侧状态经 `WorkflowPluginStatusesForUser` 按 EffectiveEnabled 输出 |
| 官方前端应用 | 浏览器宿主内，无沙箱隔离 | `web/src/lib/plugins/builtin/`（eagle、prompt-optimizer、ai-art-critique、media-conversion、user-feedback、official-asset-library、workflows、eagle-runtime）+ `web/src/services/plugin-host.ts` | 前端 `web/src/lib/plugins/plugin-registry.ts` `registerPlugin` 注册 + 后端 `plugin_management.go` `officialApplicationPolicies` 策略表 | 宿主上下文仅 storage（`pluginStorageFor`）与 `ai.text`（`requestToolResponse`，需 permissions 含 `ai.text`）；canvasNodes 贡献进 `node-registry`；特权与启停作用域由宿主 `PluginManagementView` 计算，清单不可自选 |

插件 ID 实勘字面量：前端 `EAGLE_PLUGIN_ID="eagle-asset-connector"`、`PROMPT_OPTIMIZER_PLUGIN_ID="prompt-optimizer"`、`ART_CRITIQUE_PLUGIN_ID="ai-art-critique"`、`MEDIA_CONVERSION_PLUGIN_ID="media-conversion"`、`USER_FEEDBACK_PLUGIN_ID="user-feedback"`、`OFFICIAL_ASSET_LIBRARY_PLUGIN_ID="official-asset-library"`、`RUNNINGHUB_PLUGIN_ID="runninghub-workflow-provider"`；后端短信 ID 为 `official-sms-aliyun` / `official-sms-tencent`（`backend/internal/sms/service.go`）。

## 二、plugin-packages/ 包盘点（实勘）

`ls plugin-packages` 共 248 项：119 个 `.yingce-plugin` 官方包 + 122 个同名源目录 + 7 个构建/校验脚本（build-packages.sh、package-directories.mjs、generate-catalog.mjs、generate-catalog.test.mjs、embed-documentation.mjs、package-selected.ps1、verify-payment-packages.sh）。

119 个包按包内 manifest 实际内容逐包解包分组：

| 分组 | 数量 | runtime.backend | 备注 |
|---|---|---|---|
| 协议/请求 provider | 113 | declarative | 覆盖 openai/anthropic/gemini 系及各国内外厂商、聚合站的请求协议适配 |
| 支付 | 5 | rpc | official-payment-*，backendEntry 均为 `backend/provider`，包内含多平台可执行件 |
| 工作流 | 1 | 未声明 backend（declarative 投影） | `autodl-comfyui`，apiVersion=v2，贡献 contributes.workflows |
| 短信 | 0 | — | 短信无包，bundled 清单随宿主二进制发布 |

apiVersion 分布：`yingce.plugin/v2` × 114、`yingce.plugin/v1` × 5（5 个 v1 全部为支付包）。

包盘点口径：`audit/02-plugins.md` 与本页均为实勘数 119 个 `.yingce-plugin` 包（v2×114、v1×5），已核对一致。

## 三、分发与「市场」

- **官方目录**：`backend/internal/app/protocol_plugins.go` `officialPluginPackageDir` 优先读环境变量 `CANVAS_OFFICIAL_PLUGIN_DIR`（须为可读目录），否则依次探测 `/app/plugin-packages` 与当前工作目录向上最多 8 级的 `plugin-packages`；官方包随镜像不可变发布，启动期 `inspectOfficialPluginPackage` 按 SHA-256 与文件身份缓存校验结果（`InspectPluginPackage` 仅解压清单），并经 `cachePluginPackage` 缓存进数据目录。
- **插件中心列表 = 前端 builtin + 后端已装合并**：`web/src/pages/plugins/index.tsx` 将 `listRegisteredPlugins()`（前端 builtin 注册表）与 `fetchPlugins()`（GET `/api/plugins` 返回的已装清单与状态）按 ID 合并——`byId` Map 中后端项覆盖同 ID 前端项——单一页面同时呈现协议/支付/短信/工作流/官方应用五类记录。
- **已装包下载**：`GET /plugins/:id/package`（`backend/internal/handler/plugin.go`）先做登录与可见性校验，再以 `application/zip` + attachment 响应原始 `.yingce-plugin` 字节；后端 `Service.PluginPackage` 直接读取数据目录中按 SHA-256 命名的包文件，无 fileName 时回退 `<id>.yingce-plugin`。
- **安装期自动校验即审核**：上传件经 `ParsePluginPackage` 结构校验、保留 ID 校验（官方应用/系统支付 ID 不可占用）、`LoadInstalledProviders` 可加载校验、版本/摘要一致性校验（`retainRequestPackage`），支付插件升级/卸载另有未完成订单闸（`ensurePaymentPluginLifecycle`/`ensurePaymentPluginCanBeRemoved`）；全流程无人工审核环节，"市场"即官方包目录与上传通道的并集。

## 四、与渠道体系的关系

自定义渠道不在插件体系内：浏览器端 `web/src/services/api/custom-channel-relay.ts` `channelRequest` 把上游 URL、渠道 Base URL、线格式与自定义头包装为 `X-Canvas-Upstream-*` 请求头发往登录态后端 `/ai/custom`；`backend/internal/handler/custom_proxy.go` `RegisterCustomRelayRoutes` 的 `r.Any("/ai/custom")` 负责 URL 校验、功能开关、每分钟限流与并发槽协调后转发上游。该链路不注册插件、无 manifest 参与。

两个衔接点：

1. **插件目录的 user.custom-channel 作用域**：`GET /plugins/catalog` 默认 `scope=user.custom-channel`（`backend/internal/handler/plugin.go`），`Service.PluginProviderCatalog` 把声明式 provider/workflow 贡献按该作用域投影为自定义渠道可选的模型目录——自定义渠道消费的是协议插件的清单贡献，而非其运行时。
2. **Go 侧 ModelCatalogPlugin 按 baseURL 补目录**：`backend/internal/provider/interface.go` 定义 `ModelCatalogPlugin`（只补充标准 `/models` 未暴露的厂商目录项），`backend/internal/provider/registry.go` `MatchProvider(baseURL, headers)` 按渠道地址匹配厂商；`backend/internal/app/channel_model_catalog_plugin.go` `extendChannelModelCatalog` 在标准目录之上追加条目（插件不持密钥、不自行发 HTTP，受环境变量 `ENABLE_PROVIDER_PLUGINS` 开关控制），当前仅注册 bailian 一家。
