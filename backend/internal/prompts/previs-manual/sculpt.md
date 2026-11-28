# 自由造型配方

用 `object_add` 设置 `kind:"sculpt"`、`sculpt:{version:1,parts:[...]}` 创建。`object_update` 的 `sculpt` 整体替换已有配方；先以 includeTransforms=true 读取规范化配方，再修改。沿用已有审批和 snapshotHash 并发校验。配方是 JSON 数据，不执行代码。浏览器负责真实几何和严格布尔生成，服务端校验通过不代表已经看过生成画面。

根仅允许 version、parts。每部件必填 id、shape、size。id 为 1–40 位小写字母/数字/连字符，首位字母或数字，不重复。size 是三维米制尺寸，每轴 0.005–20。position 默认 [0,0,0]，每轴绝对值不超过 20；rotation 默认 [0,0,0]，单位度，XYZ 顺序。Y 向上。部件几何居中，落地需 position.y 为半高。父坐标通过 parent 引用，尺寸不继承父部件尺寸。

通用可选字段：color 为 #rgb/#rrggbb（默认 #c2c6c8），parent 为父部件 ID，mirror 布尔值。material 为 clay/wood/straw/stone/brick/concrete/metal/glass/ceramic/plastic/rubber/fabric/paper/foliage；finish 为 matte/satin/gloss。detail:{amount:0–1,scale?:1–12,seed?:0–999999}，scatter 不支持 detail。gradient:{color,axis?:x/y/z,from?:0–1,to?:0–1}。

shape 与专用字段（其他形状不得携带这些字段）：

shape 只允许下表 11 类。圆锥／尖塔用 `cylinder` 加 `taper:1`，不是 `shape:cone`；需要自定义顶点和三角面时读 `manual=mesh`，使用独立的 `kind=mesh`。

| shape | 专用字段 |
| --- | --- |
| blob | roundness 0–1 默认 0.5（越大越方），taper -1–1 默认 0 |
| box | roundness 0–1 默认 0.1 |
| cylinder | taper -1–1 默认 0 |
| torus | 无；局部 XY 平面，size.z 管径必须小于 size.x/size.y |
| frame | border 米制宽度，大于 0 且小于短边一半；默认短边 15% |
| lathe | profile 3–32 个 [r,y] 点，各轴 0–1；y 不下降且首尾有高度差，r 不能全 0；smooth 布尔默认 true |
| tube | path 2–32 个三维点，各轴 -0.5–0.5；thickness 必填直径 0.005–最小尺寸；taper -1–1；closed 布尔默认 false，闭合至少 3 点且不能 taper，拟合后相邻点不能重合 |
| extrude | outline 3–64 个二维点，各轴 -0.5–0.5；holes 最多 4 个，各孔 3–32 点；轮廓无自交和重边，孔在外轮廓内、不相交或嵌套；roundness 0–1 |
| loft | sections 2–12 项，高度严格递增，首 y=0 尾 y=1；每项 {y,points:[[x,z]...]}（3–24 点，各轴 -0.5–0.5，无自交且有面积）或 {y,w,d,x?,z?,roundness?}（w/d 0.01–1，x/z 默认 0，截面不能超出 ±0.5）；smooth 默认 true |
| convex | points 4–48 个三维点（各轴 -0.5–0.5，非共线/共面、最小深度 0.02），或 seed 整数 0–2147483647，二选一；种子岩石 facetCount 整数 6–40 默认 16；facets 默认 true |
| scatter | item 必填 blade/tuft/flower/leaf/pebble/twig，count 必填整数 1–4000，seed 整数 0–999999 默认 0，spread ground/shell 默认 ground |

mirror 复制整个子树，关于父坐标 X=0 镜像。父链最大深度 8（根为 0），禁止父环和嵌套镜像；展开后最多 48 部件、12000 个 scatter 实例。

op 可为 subtract 或 intersect，该部件作为切割体，只作用于 parent，不显示，切面沿用父材质。切割体必须有可见父部件，不能有子部件，不能切割 scatter。多个切割按配方顺序执行。普通部件叠放不会自动焊接。所有未知字段和不适用字段拒绝。

最小示例：

```json
{"type":"object_add","id":"sculpt-cup","name":"造型杯","kind":"sculpt","sculpt":{"version":1,"parts":[{"id":"body","shape":"blob","size":[1,1,1],"position":[0,0.5,0]},{"id":"handle","shape":"torus","size":[0.6,0.6,0.1],"parent":"body","position":[0.6,0,0]}]}}
```

上述杯体没有自动生成内腔；需要空腔时显式添加 subtract 切割体。校验失败反馈字段路径，几何失败保留上一版，不得声称失败的切割已经完成。
