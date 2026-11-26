# 路由与门控全量清单

本页从 `web/src/router.tsx` 逐条枚举产品路由、管理后台子路由与门控机制，是 [03 产品模块审计](../03-product-modules.md) 的证据层；路由增删以 router.tsx 为准。

## 顶层路由表

| 路由 | 页面组件（`web/src/`） | 守卫 / 门控 |
| --- | --- | --- |
| `/dev/folders`、`/dev/previs-repro` | `pages/dev/`（FolderPreviewLab、PrevisReproLab） | 仅 `import.meta.env.DEV` 摇树，**无登录守卫**（DEV 实验页） |
| `/test-voice-recording` | `pages/test-voice-recording.tsx` | `RequireAuth` + DEV 摇树 |
| `/login` `/register` `/forgot-password` | `pages/auth/`（LoginPage、RegisterPage、ForgotPasswordPage） | 无（AuthScene 承载） |
| `/share/canvas/:token` | `pages/canvas/shared.tsx`（SharedCanvasPage） | 无登录守卫（分享语义） |
| `/ark-portrait/result` | `pages/ark-portrait-result`（ArkPortraitResultPage） | 无（H5 真人认证回跳，多在未登录手机打开） |
| `/`、`/create` | `pages/create`（CreatePage） | 工作区登录守卫 |
| `/ecommerce` | `pages/ecommerce`（EcommercePage） | `RequireAuth` |
| `/toolbox/image`、`/toolbox/video` | `pages/workbench/` | `RequireAuth` + 功能开关门控 |
| `/tasks` | `pages/tasks` | `RequireAuth` |
| `/assets` | `pages/assets`（AssetsPage） | `RequireAuth` |
| `/teams`、`/teams/:teamId` | `pages/teams/` | `RequireAuth`；子路由 index 重定向到 canvas |
| `/teams/:teamId/create`、`/teams/:teamId/wallet` | TeamMembersPage 同组 | `RequireFeature feature="creditsEnabled"` |
| `/skills` | `pages/skills`（SkillsPage） | `RequireAuth` |
| `/skills/:skillId/edit` | SkillWorkbenchPage | `RequireSkillAdmin` |
| `/plugins`、`/plugins/eagle` | `pages/plugins/` | `RequireAuth` + 插件启用态门控 |
| `/wallet` | `router.tsx` 渲染 `RequireAuth{null}` | **空壳路由**（钱包功能在全局钱包弹窗） |
| `/settings` | `pages/settings`（SettingsPage） | `RequireAuth` |
| `/referral` | `pages/referral`（ReferralPage） | `RequireAuth` |
| `/projects` 系（index、new、/:projectId、/:projectId/e/:episodeId） | `pages/projects/` + `studio/` | `RequireAuth`；`/projects/:projectId/*` 兜底重定向回上级 |
| `/canvas`、`/canvas/:id` | `pages/canvas/`（CanvasPage、CanvasProjectPage + CanvasRefreshShell） | `RequireAuth` |
| `/admin` 系 | `pages/admin/`（AdminPage 懒加载入口） | `RequireAuth`（管理权限由后端接口校验） |
| `*` | `pages/not-found`（NotFound） | — |

## 管理后台子路由全枚举（`/admin` 段）

按域分组，符号为 router.tsx 内懒加载组件：

- **流量与日志**：index（AnalyticsPage）、`traffic`（TrafficPage）、`logs`（LogsPage）、`login-logs`（LoginLogsPage）、`email-logs`（EmailLogsPage）、`feedbacks`（AdminFeedbacksPage）
- **用户与凭证**：`users`（UsersPage）、`user-groups`（UserGroupsPage）、`redemption-codes`（RedemptionCodesPage）、`referral`（AdminReferralPage）、`credit-operations`（CreditOperationsPage）
- **渠道与模型**：`channels`（ChannelsPage）、`models`（LogicalModelsPage，注释：关闭前台模型只影响用户端来源）
- **电商域**：`ecommerce`（重定向 → `/admin/ecommerce/agent`）、`ecommerce/agent`（EcommerceAgentConfigPage）、`ecommerce/library`（EcommerceManagementPage）、`ecommerce/library/skills`、`ecommerce/library/categories`
- **素材与技能**：`official-assets`（AdminOfficialAssetsPage）、`official-asset-recommendations`（AdminOfficialAssetRecommendationsPage）、`pose-library`（PoseLibraryAdminPage）、`skills`（AdminSkillsPage）、`skills/:skillId/edit`（AdminSkillWorkbenchPage）、`skill-curation`（SkillCurationPage）、`resources`（StorageResourcesPage）
- **插件与 Agent 接入**：`plugins`（AdminPluginsPage）、`mcp`（AdminMCPPage）、`agent-access`（AgentAccessPage）、`agent-access/authorize`、`agent-lessons`（AgentLessonsPage）、`workbench-tools`（WorkbenchToolsPage）
- **支付与运营**：`payments`（AdminPaymentsPage）、`prompt-templates`（StoryboardPromptsPage）、`announcements`（AnnouncementsPage）、`banner-announcements`（BannerAnnouncementsPage）
- **设置域**：`settings`（重定向 → runtime-policy）、`settings/appearance`、`settings/drawing-engine`、`settings/runtime-policy`、`settings/features`（FeatureAvailabilityPage）、`settings/access`（AccessSettingsPage）、`settings/email`（EmailSettingsPage）、`settings/storage`（StorageSettingsPage）、`settings/ark-private-assets`、`settings/response-interception`（ResponseInterceptionSettingsPage）、`settings/third-party`（ThirdPartySettingsPage）、`settings/system-update`（SystemUpdatePage）、`settings/system-performance`（SystemPerformancePage）

## 旧地址重定向清单（`Navigate`）

| 旧地址 | 去向 |
| --- | --- |
| `/admin/ecommerce` | `/admin/ecommerce/agent` |
| `/admin/ecommerce/skills` | `/admin/ecommerce/library/skills` |
| `/admin/ecommerce/categories` | `/admin/ecommerce/library/categories` |
| `/admin/ecommerce-knowledge` | `/admin/ecommerce/library` |
| `/admin/ecommerce-skills` | `/admin/ecommerce/library/skills` |
| `/admin/ecommerce-categories` | `/admin/ecommerce/library/categories` |
| `/admin/storyboard-prompts` | `/admin/prompt-templates` |
| `/admin/settings` | `/admin/settings/runtime-policy` |
| `/admin/settings/concurrency` | `/admin/settings/runtime-policy` |
| `/admin/settings/libtv` | `/admin/settings/third-party` |
| `/projects/:projectId/*` | 上级项目路由（`relative="path"`） |

## 守卫与门控机制

| 机制 | 符号 | 作用范围 |
| --- | --- | --- |
| 登录守卫 | `RequireAuth`（router.tsx） | 工作区全部业务路由；admin 仅有登录守卫，管理权限由后端接口校验 |
| 功能开关 | `RequireFeature`（router.tsx） | `/toolbox/image` 等工具台；团队空间的 `creditsEnabled`（create、wallet 子路由）；后端短剧域路由同机制（`FeatureShortDrama`） |
| 技能管理员 | `RequireSkillAdmin` | `/skills/:skillId/edit` |
| 插件启用态 | 路由元素内门控 | `/plugins/eagle` 等（`requireEnabledPlugin` 对应前端启用判定） |
| DEV 摇树 | `import.meta.env.DEV` | `/dev/folders`、`/dev/previs-repro`、`/test-voice-recording` 不进生产构建 |

## 审计观察（现状事实）

1. `/wallet` 是 `RequireAuth` 包裹的 `null` 空壳，且被流量统计排除；钱包入口实际是全局钱包弹窗。
2. `/dev/folders` 与 `/dev/previs-repro` 两个实验路由**没有登录守卫**（仅 DEV 构建摇树）；生产构建不可达，但源码层无守卫语义。
3. 管理后台无独立前端管理员守卫组件——页面级只有 `RequireAuth`，真正的权限判定在后端接口（与「前端隐藏按钮不算权限」的约定一致）。
4. 旧地址重定向共 10 条 `Navigate`（电商域 6 条 + 设置域 4 条），全部集中在 router.tsx 一处维护。
