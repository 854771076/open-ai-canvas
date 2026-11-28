# Sculpt 内核来源

从 `waoowaoo` 提交 `71a7d74d49287ee797722edd331967494047a2ab` 的 `vendor/cozyclay/src/` 提取配方和几何依赖闭包。

许可：**AGPL-3.0-or-later**。保留 LICENSE、LICENSING.md、THIRD_PARTY_NOTICES.md 和 upstream.json；本目录衍生修改同样按该许可提供。

宿主修改：布尔失败通过 diagnostics 数组提供结构化诊断，不覆盖全局 console；生成失败释放已登记的几何和材质；极大旋转数值避免舍入溢出，边框按存储精度校验；按宿主 Prettier 格式化。上游英文注释作为原始源码保留，新增注释使用中文。
