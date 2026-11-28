import { Canvas } from "@react-three/fiber";
import { Bounds, OrbitControls } from "@react-three/drei";
import { Alert, Button, Checkbox, Input, InputNumber, Select, Tree } from "antd";
import { useEffect, useRef, useState } from "react";
import type { Group } from "three";
import { AppModal } from "@/components/ui/product/app-modal";
import type { PrevisSculptPart, PrevisSculptRecipe, PrevisSculptShape } from "@/types/previs-sculpt";
import { buildPrevisSculptGroup, disposePrevisSculptGroup, normalizePrevisSculptRecipe, patchPrevisSculptRecipe } from "@/lib/canvas/previs/previs-sculpt";
import "./previs-modeling.css";
import { PrevisSculptFields } from "./previs-sculpt-fields";

const SHAPES: [PrevisSculptShape, string][] = [
    ["blob", "软团块"],
    ["box", "方块"],
    ["cylinder", "圆柱"],
    ["torus", "圆环"],
    ["frame", "边框"],
    ["lathe", "旋转轮廓"],
    ["tube", "曲线管道"],
    ["extrude", "轮廓挤出"],
    ["loft", "截面放样"],
    ["convex", "凸包岩石"],
    ["scatter", "实例散布"],
];
const MATERIALS = ["clay", "wood", "straw", "stone", "brick", "concrete", "metal", "glass", "ceramic", "plastic", "rubber", "fabric", "paper", "foliage"];
const MATERIAL_LABELS: Record<string, string> = {
    clay: "黏土",
    wood: "木材",
    straw: "稻草",
    stone: "石材",
    brick: "砖石",
    concrete: "混凝土",
    metal: "金属",
    glass: "玻璃",
    ceramic: "陶瓷",
    plastic: "塑料",
    rubber: "橡胶",
    fabric: "织物",
    paper: "纸张",
    foliage: "叶片",
};
const SCALAR_LABELS = { roundness: "圆方程度", taper: "收尖程度", border: "边框厚度（米）", thickness: "管径（米）", count: "实例数量", seed: "造型种子", facetCount: "岩石面数" };
export function previsSculptShapeExample(shape: PrevisSculptShape, id = "body"): PrevisSculptPart {
    const base: PrevisSculptPart = { id, shape, size: [1, 1, 1], position: [0, 0.5, 0], color: "#8795a5" };
    switch (shape) {
        case "torus":
            return { ...base, size: [1, 1, 0.15] };
        case "frame":
            return { ...base, size: [1, 1, 0.15], border: 0.1 };
        case "lathe":
            return {
                ...base,
                profile: [
                    [0, 0],
                    [1, 0.1],
                    [0.6, 0.6],
                    [0.8, 1],
                ],
                smooth: true,
            };
        case "tube":
            return {
                ...base,
                path: [
                    [-0.4, -0.4, 0],
                    [0.4, 0, 0],
                    [-0.4, 0.4, 0],
                ],
                thickness: 0.15,
            };
        case "extrude":
            return {
                ...base,
                size: [1, 1, 0.2],
                outline: [
                    [-0.5, -0.5],
                    [0.5, -0.5],
                    [0, 0.5],
                ],
            };
        case "loft":
            return {
                ...base,
                sections: [
                    { y: 0, w: 0.5, d: 0.5 },
                    { y: 0.5, w: 1, d: 1 },
                    { y: 1, w: 0.5, d: 0.5 },
                ],
            };
        case "convex":
            return { ...base, seed: 1, facetCount: 16 };
        case "scatter":
            return { ...base, count: 100, item: "blade", seed: 1 };
        default:
            return base;
    }
}
const DEFAULT: PrevisSculptRecipe = { version: 1, parts: [previsSculptShapeExample("blob")] };

export function PrevisSculptEditor({ initial, name: initialName, onClose, onConfirm }: { initial?: PrevisSculptRecipe; name?: string; onClose: () => void; onConfirm: (recipe: PrevisSculptRecipe, name: string) => void }) {
    const [recipe, setRecipe] = useState(() => {
            const result = normalizePrevisSculptRecipe(initial ?? DEFAULT);
            return result.ok ? result.recipe : DEFAULT;
        }),
        [name, setName] = useState(initialName ?? "自由造型");
    const [selected, setSelected] = useState(recipe.parts[0]?.id ?? ""),
        [shape, setShape] = useState<PrevisSculptShape>("box");
    const [error, setError] = useState(""),
        [json, setJson] = useState(JSON.stringify(recipe, null, 2));
    const [group, setGroup] = useState<Group>();
    const owned = useRef(new Set<Group>());
    const pending = useRef<Group | undefined>(undefined);
    const [ready, setReady] = useState(false);
    const fileInput = useRef<HTMLInputElement>(null);
    const alive = useRef(true);
    useEffect(() => {
        alive.current = true;
        return () => {
            alive.current = false;
        };
    }, []);
    const install = (input: unknown) => {
        const result = normalizePrevisSculptRecipe(input);
        if (!result.ok) {
            setError(`${result.path}: ${result.error}`);
            return false;
        }
        try {
            // 严格生成成功后才替换有效配方，校验或布尔失败继续显示旧模型。
            const next = buildPrevisSculptGroup(result.recipe);
            owned.current.add(next);
            pending.current = next;
            setGroup(next);
            setRecipe(result.recipe);
            setJson(JSON.stringify(result.recipe, null, 2));
            setError("");
            setReady(true);
            return true;
        } catch (reason) {
            setError(reason instanceof Error ? reason.message : "几何生成失败");
            return false;
        }
    };
    useEffect(() => {
        const result = normalizePrevisSculptRecipe(initial ?? DEFAULT);
        if (!result.ok) {
            setError(`${result.path}: ${result.error}`);
            return;
        }
        let next: Group;
        try {
            next = buildPrevisSculptGroup(result.recipe);
            owned.current.add(next);
            pending.current = next;
            setGroup(next);
            setReady(true);
        } catch (reason) {
            setError(reason instanceof Error ? reason.message : "几何生成失败");
            return;
        }
        return () => {
            disposePrevisSculptGroup(next);
            owned.current.delete(next);
        };
    }, []);
    useEffect(() => {
        if (!group || group !== pending.current) return;
        for (const previous of owned.current) {
            if (previous === group) continue;
            disposePrevisSculptGroup(previous);
            owned.current.delete(previous);
        }
    }, [group]);
    useEffect(
        () => () => {
            owned.current.forEach(disposePrevisSculptGroup);
            owned.current.clear();
        },
        [],
    );
    const part = recipe.parts.find((p) => p.id === selected);
    const update = (patch: Partial<PrevisSculptPart> & { parent?: string | undefined }) => {
        if (!part) return;
        install({ ...recipe, parts: recipe.parts.map((p) => (p.id === part.id ? { ...p, ...patch } : p)) });
    };
    type PartTreeNode = { key: string; title: string; children: PartTreeNode[] };
    const tree: (parent?: string) => PartTreeNode[] = (parent) => recipe.parts.filter((p) => p.parent === parent).map((p) => ({ key: p.id, title: `${p.id} · ${SHAPES.find(([s]) => s === p.shape)?.[1]}${p.op ? "（切割）" : ""}`, children: tree(p.id) }));
    const vector = (key: "size" | "position" | "rotation", label: string) => (
        <label className="previs-modeling__field">
            {label}
            <span>
                {[0, 1, 2].map((axis) => (
                    <InputNumber
                        key={axis}
                        aria-label={`${label} ${"XYZ"[axis]}`}
                        value={(part?.[key] ?? [0, 0, 0])[axis]}
                        step={key === "rotation" ? 5 : 0.05}
                        onChange={(v) => {
                            if (!part || v === null) return;
                            const value = [...(part[key] ?? [0, 0, 0])] as [number, number, number];
                            value[axis] = v;
                            update({ [key]: value });
                        }}
                        style={{ width: "32%" }}
                    />
                ))}
            </span>
        </label>
    );
    return (
        <AppModal open flush width="min(1180px, 96vw)" title={null} footer={null} onCancel={onClose} mask={{ closable: false }}>
            <div className="previs-modeling" data-canvas-no-zoom data-canvas-wheel-scroll>
                <header className="previs-modeling__header">
                    <h2>自由造型</h2>
                    <span>配方建模 · 米制尺寸 · 旋转单位度</span>
                </header>
                <div className="previs-modeling__toolbar">
                    <input
                        ref={fileInput}
                        hidden
                        type="file"
                        accept=".json,application/json"
                        onChange={(event) => {
                            const file = event.target.files?.[0];
                            event.target.value = "";
                            if (!file) return;
                            if (file.size > 1024 * 1024) {
                                setError("配方文件不能超过 1 MiB");
                                return;
                            }
                            void file
                                .text()
                                .then((text) => {
                                    if (!alive.current) return;
                                    try {
                                        install(JSON.parse(text));
                                    } catch {
                                        setError("配方文件不是有效 JSON");
                                    }
                                })
                                .catch(() => {
                                    if (alive.current) setError("配方文件读取失败");
                                });
                        }}
                    />
                    <Button onClick={() => fileInput.current?.click()}>导入配方 JSON</Button>
                    <Button
                        onClick={() => {
                            const url = URL.createObjectURL(new Blob([JSON.stringify(recipe, null, 2)], { type: "application/json" }));
                            const link = document.createElement("a");
                            link.href = url;
                            link.download = `${name.replace(/[\\/:*?"<>|]/g, "_") || "自由造型"}.json`;
                            link.click();
                            setTimeout(() => URL.revokeObjectURL(url), 1000);
                        }}
                    >
                        导出配方
                    </Button>
                    <Select aria-label="新增部件形状" value={shape} options={SHAPES.map(([value, label]) => ({ value, label }))} onChange={setShape} />
                    <Button
                        onClick={() => {
                            let index = 1;
                            while (recipe.parts.some((p) => p.id === `part-${index}`)) index++;
                            const next = previsSculptShapeExample(shape, `part-${index}`);
                            if (install({ ...recipe, parts: [...recipe.parts, next] })) setSelected(next.id);
                        }}
                    >
                        添加部件示例
                    </Button>
                    <Button
                        danger
                        disabled={!part || recipe.parts.length <= 1}
                        onClick={() => {
                            if (!part) return;
                            const result = patchPrevisSculptRecipe(recipe, [{ remove: part.id }]);
                            if (!result.ok) setError(`${result.path}: ${result.error}`);
                            else if (install(result.recipe)) setSelected(result.recipe.parts[0].id);
                        }}
                    >
                        删除部件及子部件
                    </Button>
                </div>
                {error ? (
                    <Alert
                        type="error"
                        showIcon
                        title={error}
                        action={
                            <Button size="small" onClick={() => install(recipe)}>
                                恢复有效配方
                            </Button>
                        }
                    />
                ) : null}
                <div className="previs-modeling__body">
                    <div className="previs-modeling__preview">
                        {/* 与网格编辑器一致，忽略弹窗入场缩放，按真实布局尺寸测量预览。 */}
                        <Canvas resize={{ offsetSize: true }} camera={{ position: [3, 2, 4], fov: 45 }}>
                            <ambientLight intensity={1.7} />
                            <directionalLight position={[4, 6, 4]} intensity={2.5} />
                            {group ? (
                                <Bounds fit clip observe margin={1.4}>
                                    <primitive object={group} dispose={null} />
                                </Bounds>
                            ) : null}
                            <OrbitControls makeDefault />
                        </Canvas>
                    </div>
                    <aside className="previs-modeling__inspector">
                        <label className="previs-modeling__field">
                            名称
                            <Input value={name} maxLength={160} onChange={(e) => setName(e.target.value)} />
                        </label>
                        <Tree aria-label="部件树" treeData={tree()} defaultExpandAll selectedKeys={[selected]} onSelect={(keys) => setSelected(String(keys[0] ?? ""))} />
                        {part ? (
                            <>
                                <label className="previs-modeling__field">
                                    部件形状
                                    <Select
                                        value={part.shape}
                                        options={SHAPES.map(([value, label]) => ({ value, label }))}
                                        onChange={(value) => {
                                            const { id: _id, ...fields } = previsSculptShapeExample(value, part.id);
                                            const result = patchPrevisSculptRecipe(recipe, [{ set: part.id, ...fields, position: part.position, color: part.color }]);
                                            if (result.ok) install(result.recipe);
                                            else setError(`${result.path}: ${result.error}`);
                                        }}
                                    />
                                    <span className="previs-modeling__note">更换形状使用该形状的默认尺寸与控制点</span>
                                </label>
                                <label className="previs-modeling__field">
                                    父部件
                                    <Select
                                        aria-label="父部件"
                                        value={part.parent ?? ""}
                                        options={[{ value: "", label: "无（根部件）" }, ...recipe.parts.filter((p) => p.id !== part.id).map((p) => ({ value: p.id, label: p.id }))]}
                                        onChange={(v) => update({ parent: v || undefined })}
                                    />
                                </label>
                                <Checkbox checked={part.mirror ?? false} onChange={(e) => update({ mirror: e.target.checked })}>
                                    镜像整个子树
                                </Checkbox>
                                {vector("size", "尺寸（米）")}
                                {vector("position", "位置（米）")}
                                {vector("rotation", "旋转（度）")}
                                <label className="previs-modeling__field">
                                    颜色
                                    <input aria-label="部件颜色" type="color" value={part.color ?? "#c2c6c8"} onChange={(e) => update({ color: e.target.value })} />
                                </label>
                                <label className="previs-modeling__field">
                                    材质
                                    <Select value={part.material ?? "clay"} options={MATERIALS.map((value) => ({ value, label: MATERIAL_LABELS[value] }))} onChange={(value) => update({ material: value as PrevisSculptPart["material"] })} />
                                </label>
                                <label className="previs-modeling__field">
                                    表面
                                    <Select
                                        value={part.finish ?? ""}
                                        options={[
                                            { value: "", label: "材质默认" },
                                            { value: "matte", label: "哑光" },
                                            { value: "satin", label: "缎光" },
                                            { value: "gloss", label: "亮光" },
                                        ]}
                                        onChange={(value) => update({ finish: (value || undefined) as PrevisSculptPart["finish"] })}
                                    />
                                </label>
                                <label className="previs-modeling__field">
                                    切割父部件
                                    <Select
                                        value={part.op ?? ""}
                                        options={[
                                            { value: "", label: "显示部件" },
                                            { value: "subtract", label: "从父部件减去" },
                                            { value: "intersect", label: "保留与父部件交集" },
                                        ]}
                                        onChange={(value) => update({ op: (value || undefined) as PrevisSculptPart["op"] })}
                                    />
                                </label>
                                {(["roundness", "taper", "border", "thickness", "count", "seed", "facetCount"] as const)
                                    .filter((key) => part[key] !== undefined)
                                    .map((key) => (
                                        <label className="previs-modeling__field" key={key}>
                                            {SCALAR_LABELS[key]}
                                            <InputNumber
                                                value={part[key]}
                                                step={key === "count" || key === "seed" || key === "facetCount" ? 1 : 0.05}
                                                onChange={(value) => {
                                                    if (value !== null) update({ [key]: value });
                                                }}
                                            />
                                        </label>
                                    ))}
                                <PrevisSculptFields part={part} onChange={update} />
                            </>
                        ) : null}
                        <details>
                            <summary>高级配方 JSON（孔洞、细节、渐变）</summary>
                            <Input.TextArea aria-label="高级配方 JSON" rows={12} value={json} onChange={(e) => setJson(e.target.value)} />
                            <Button
                                onClick={() => {
                                    try {
                                        install(JSON.parse(json));
                                    } catch {
                                        setError("JSON 格式无效，已保留原模型");
                                    }
                                }}
                            >
                                校验并应用配方
                            </Button>
                        </details>
                    </aside>
                </div>
                <footer className="previs-modeling__footer">
                    <span>{recipe.parts.length} 部件 · 无效修改保留上次模型</span>
                    <Button onClick={onClose}>取消</Button>
                    <Button
                        type="primary"
                        disabled={!ready || !name.trim() || Boolean(error) || json !== JSON.stringify(recipe, null, 2)}
                        onClick={() => {
                            onConfirm(recipe, name.trim());
                        }}
                    >
                        确认造型
                    </Button>
                </footer>
            </div>
        </AppModal>
    );
}
