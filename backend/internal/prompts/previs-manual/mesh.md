# 三角网格建模

没有合适的目录道具或模型时，自己建模，不要因为缺少素材而中止。规则部件、管道、旋转轮廓、布尔切割优先 `manual=sculpt`；需要精确轮廓、镂空结构或直接控制拓扑时用 `kind=mesh`。不要把 mesh、cone、pyramid 当作 Sculpt 的 shape。

## 创建与修改

- 用 `previs_apply_patch` 的 `object_add kind=mesh` 提交 `mesh:{version:1,vertices,triangles,colors}`。这是受校验的结构化数据，不是脚本。
- vertices：3–20000 个 `[x,y,z]`，对象局部空间，单位米，Y 向上，每轴绝对值不超过 10000。建议底面放在 y=0，便于 position.y=0 贴地。无隐式归一化和缩放。
- triangles：1–40000 个 `[a,b,c]`，索引从 0 开始且必须引用已有顶点。按从物体外侧看逆时针排列。不能重复索引或生成零面积三角面；需要实体时把底面、侧面、顶面一起闭合。允许平面网格。
- colors：与 triangles 等长，每面一个 `#rrggbb`。改色需要修改 colors，对象 color 不覆盖面颜色。
- 根只允许 version、vertices、triangles、colors，全部必填。每项补丁最多 32 个操作，画布内所有场景合计仍受 1 MiB 上限约束；从低面数轮廓开始，不要一次输出数万面。
- 修改前用 `previs_scene_read(sceneId,objectIds:[目标ID],includeTransforms:true)` 获取最新 snapshotHash 和 `scene.spatialLayout.objects[].mesh`。普通摘要只有 meshSummary，不含完整数组。
- 用 `object_update id=原ID mesh=修改后的完整拓扑` 整体替换网格；对象 ID、变换、关键帧和父子关系保留。省略 mesh 时只改对象属性；null 不代表删除，会报错。顶点、边、面编辑可通过改坐标、拆面、增加挤出的顶点与侧面、合并索引等方式完成；每次都提交完整有效拓扑，更新面数时同步 colors。
- 拓扑只适用于 kind=mesh，不会隐式把已有道具或 GLB 改成网格。用户可在检查器「复制为可编辑网格」手动转换其他静态对象；转换出的 GLB 仍是模型素材。Agent 对自己创建的 kind=mesh 可直接读回并继续编辑。

## 示例：从零建立方锥

```json
{"type":"object_add","id":"mesh-spire","name":"三角网格尖塔","kind":"mesh","position":[0,0,0],"mesh":{"version":1,"vertices":[[-1,0,-1],[1,0,-1],[1,0,1],[-1,0,1],[0,3,0]],"triangles":[[0,1,2],[0,2,3],[0,4,1],[1,4,2],[2,4,3],[3,4,0]],"colors":["#63758b","#63758b","#b8c7d8","#8ba0b8","#71869f","#a2b4c8"]}}
```

示例底面朝 -Y，四个侧面朝外。真实物体应先规划尺寸、轮廓和主要结构，再按需要补细节；不要以一个基础体替代用户明确要求的完整模型。

## 反馈与验收

沿用 snapshotHash、用户归属和审批边界。顶点、索引、颜色或退化问题会以字段错误交回模型，修改后重试，最多三次；不要原样重复失败调用。审批前不会修改场景。

创建后读回核对数量、尺寸、位置，调用 `previs_preview` 检查白模视频，再按实际画面修正。数据校验通过不代表已经看过画面。视口可显示自建网格，也能打开「编辑顶点／边／面」继续操作；手动保存会导出独立 GLB 素材，同时保留场景里的原网格和对象 ID，供 Agent 继续修改。
