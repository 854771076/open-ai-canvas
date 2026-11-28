import { Canvas, useThree, type ThreeEvent } from "@react-three/fiber";
import { OrbitControls, TransformControls } from "@react-three/drei";
import { Alert, Button, Input, InputNumber, Segmented } from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { BufferGeometry, Color, DoubleSide, Float32BufferAttribute, Matrix4, Object3D, Vector3, type Camera } from "three";
import { AppModal } from "@/components/ui/product/app-modal";
import {
    buildPrevisMeshGeometry,
    deletePrevisMeshFaces,
    extrudePrevisMeshFaces,
    mergePrevisMeshVertices,
    previsMeshBounds,
    previsMeshEdges,
    previsMeshSelectedVertices,
    previsMeshSelectionCenter,
    splitPrevisMeshEdges,
    transformPrevisMesh,
    type PrevisEditableMesh,
    type PrevisMeshSelection,
} from "@/lib/canvas/previs/previs-mesh";
import { createPrevisMeshSeed } from "@/lib/canvas/previs/previs-mesh-conversion";
import "./previs-modeling.css";

type EditorProps = {
    initial: PrevisEditableMesh;
    name: string;
    replacing: boolean;
    onClose: () => void;
    onSave: (mesh: PrevisEditableMesh, name: string, signal: AbortSignal) => Promise<void>;
};

export function PrevisMeshEditor({ initial, name: initialName, replacing, onClose, onSave }: EditorProps) {
    const [mesh, setMesh] = useState(initial),
        [name, setName] = useState(initialName);
    const [frameMesh, setFrameMesh] = useState(initial);
    const [past, setPast] = useState<PrevisEditableMesh[]>([]),
        [future, setFuture] = useState<PrevisEditableMesh[]>([]);
    const [selection, setSelection] = useState<PrevisMeshSelection>({ mode: "vertex", indices: [] });
    const [tool, setTool] = useState<"translate" | "rotate" | "scale">("translate");
    const [error, setError] = useState("");
    const [distance, setDistance] = useState(0.2),
        [axis, setAxis] = useState<[number, number, number]>([0, 0, 0]);
    const [saving, setSaving] = useState(false),
        [boxSelect, setBoxSelect] = useState(false);
    const [rectangle, setRectangle] = useState<{ x: number; y: number; left: number; top: number; width: number; height: number } | null>(null);
    const preview = useRef<HTMLDivElement>(null),
        camera = useRef<Camera | null>(null),
        meshRef = useRef(mesh);
    meshRef.current = mesh;
    const saveController = useRef<AbortController | null>(null);
    useEffect(
        () => () => {
            saveController.current?.abort();
            saveController.current = null;
        },
        [],
    );
    const commit = useCallback((next: PrevisEditableMesh, base = meshRef.current) => {
        if (next === base) return;
        setPast((items) => [...items.slice(-29), base]);
        setFuture([]);
        setMesh(next);
        setError("");
    }, []);
    const operate = (fn: () => PrevisEditableMesh, clear = true) => {
        try {
            commit(fn());
            if (clear) setSelection((s) => ({ ...s, indices: [] }));
        } catch (reason) {
            setError(reason instanceof Error ? reason.message : "网格操作失败");
        }
    };
    const selectedVertices = previsMeshSelectedVertices(mesh, selection);
    const undo = () => {
        const previous = past.at(-1);
        if (!previous) return;
        setFuture((items) => [...items, mesh]);
        setPast(past.slice(0, -1));
        setMesh(previous);
        setSelection((s) => ({ ...s, indices: [] }));
        setError("");
    };
    const redo = () => {
        const next = future.at(-1);
        if (!next) return;
        setPast((items) => [...items, mesh]);
        setFuture(future.slice(0, -1));
        setMesh(next);
        setSelection((s) => ({ ...s, indices: [] }));
        setError("");
    };
    const save = async () => {
        if (!name.trim() || saveController.current) return;
        // 每次保存单独持有取消信号，避免 StrictMode 重放挂载清理后复用已取消的控制器。
        const controller = new AbortController();
        saveController.current = controller;
        setSaving(true);
        setError("");
        try {
            await onSave(mesh, name.trim(), controller.signal);
        } catch (reason) {
            if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : "网格保存失败");
        } finally {
            if (saveController.current === controller) {
                saveController.current = null;
                setSaving(false);
            }
        }
    };
    const pick = useCallback((index: number, additive: boolean) => setSelection((s) => ({ ...s, indices: additive ? (s.indices.includes(index) ? s.indices.filter((i) => i !== index) : [...s.indices, index]) : [index] })), []);
    const selectBox = (rect: NonNullable<typeof rectangle>) => {
        if (!camera.current || !preview.current) return;
        const width = preview.current.clientWidth,
            height = preview.current.clientHeight;
        const candidates =
            selection.mode === "vertex"
                ? mesh.vertices.map((v) => new Vector3(...v))
                : selection.mode === "edge"
                  ? previsMeshEdges(mesh).map(([a, b]) => new Vector3(...mesh.vertices[a]).add(new Vector3(...mesh.vertices[b])).multiplyScalar(0.5))
                  : mesh.triangles.map((t) => t.reduce((p, i) => p.add(new Vector3(...mesh.vertices[i])), new Vector3()).divideScalar(3));
        const indices: number[] = [];
        candidates.forEach((p, i) => {
            p.project(camera.current!);
            const x = ((p.x + 1) * width) / 2,
                y = ((1 - p.y) * height) / 2;
            if (p.z >= -1 && p.z <= 1 && x >= rect.left && x <= rect.left + rect.width && y >= rect.top && y <= rect.top + rect.height) indices.push(i);
        });
        setSelection({ ...selection, indices });
    };
    return (
        <AppModal open flush width="min(1180px, 96vw)" title={null} footer={null} onCancel={onClose} mask={{ closable: false }} closable={!saving} keyboard={!saving}>
            <div className="previs-modeling" aria-busy={saving} data-canvas-no-zoom data-canvas-wheel-scroll>
                <header className="previs-modeling__header">
                    <h2>三角网格建模</h2>
                    <span>
                        {mesh.vertices.length} 顶点 · {mesh.triangles.length} 三角面
                    </span>
                </header>
                <div className="previs-modeling__toolbar" inert={saving}>
                    <Segmented
                        aria-label="网格选择模式"
                        value={selection.mode}
                        options={[
                            { label: "顶点", value: "vertex" },
                            { label: "边", value: "edge" },
                            { label: "面", value: "face" },
                        ]}
                        onChange={(mode) => setSelection({ mode: mode as PrevisMeshSelection["mode"], indices: [] })}
                    />
                    <Segmented
                        aria-label="网格变换工具"
                        value={tool}
                        options={[
                            { label: "移动", value: "translate" },
                            { label: "旋转", value: "rotate" },
                            { label: "缩放", value: "scale" },
                        ]}
                        onChange={(value) => setTool(value as typeof tool)}
                    />
                    <button type="button" className="previs-modeling__toggle" aria-pressed={boxSelect} onClick={() => setBoxSelect(!boxSelect)}>
                        框选穿透
                    </button>
                    <Button disabled={!past.length || saving} onClick={undo}>
                        撤销
                    </Button>
                    <Button disabled={!future.length || saving} onClick={redo}>
                        重做
                    </Button>
                    <Button onClick={() => setFrameMesh(mesh)}>适配模型</Button>
                </div>
                {error ? <Alert type="error" showIcon title={error} /> : null}
                <div className="previs-modeling__body" inert={saving}>
                    <div className="previs-modeling__preview" ref={preview}>
                        {/* 弹窗入场会缩放外壳，使用布局尺寸，避免将动画起始的 20% 当成固定绘制尺寸。 */}
                        <Canvas resize={{ offsetSize: true }} camera={{ position: [3, 2, 4], fov: 45 }} gl={{ antialias: true }}>
                            <color attach="background" args={["#e9edf0"]} />
                            <ambientLight intensity={1.7} />
                            <directionalLight position={[4, 6, 4]} intensity={2.5} />
                            <MeshCanvas
                                mesh={mesh}
                                initial={frameMesh}
                                selection={selection}
                                tool={tool}
                                onPick={pick}
                                onPreview={setMesh}
                                onCommit={commit}
                                onError={setError}
                                onCamera={(value) => {
                                    camera.current = value;
                                }}
                            />
                        </Canvas>
                        {boxSelect ? (
                            <div
                                className="previs-modeling__box-layer"
                                onPointerDown={(event) => {
                                    if (event.button !== 0 || saving) return;
                                    event.preventDefault();
                                    event.currentTarget.setPointerCapture(event.pointerId);
                                    const b = event.currentTarget.getBoundingClientRect(),
                                        x = event.clientX - b.left,
                                        y = event.clientY - b.top;
                                    setRectangle({ x, y, left: x, top: y, width: 0, height: 0 });
                                }}
                                onPointerMove={(event) => {
                                    if (!rectangle) return;
                                    const b = event.currentTarget.getBoundingClientRect(),
                                        x = event.clientX - b.left,
                                        y = event.clientY - b.top;
                                    setRectangle({ ...rectangle, left: Math.min(rectangle.x, x), top: Math.min(rectangle.y, y), width: Math.abs(x - rectangle.x), height: Math.abs(y - rectangle.y) });
                                }}
                                onPointerUp={(event) => {
                                    if (rectangle) selectBox(rectangle);
                                    setRectangle(null);
                                    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
                                }}
                                onPointerCancel={() => setRectangle(null)}
                            >
                                {rectangle ? <div className="previs-modeling__rectangle" style={{ left: rectangle.left, top: rectangle.top, width: rectangle.width, height: rectangle.height }} /> : null}
                            </div>
                        ) : null}
                        <div className="previs-modeling__hint">单击选择 · Shift 多选 · 拖动空白旋转 · 滚轮缩放</div>
                    </div>
                    <aside className="previs-modeling__inspector">
                        <label>
                            名称
                            <Input value={name} maxLength={160} onChange={(e) => setName(e.target.value)} />
                        </label>
                        <p>
                            已选 {selection.indices.length} 个{selection.mode === "vertex" ? "顶点" : selection.mode === "edge" ? "边" : "面"}（{selectedVertices.length} 顶点）
                        </p>
                        <Button
                            block
                            onClick={() => setSelection({ ...selection, indices: Array.from({ length: selection.mode === "vertex" ? mesh.vertices.length : selection.mode === "edge" ? previsMeshEdges(mesh).length : mesh.triangles.length }, (_, i) => i) })}
                        >
                            全选
                        </Button>
                        <Button block onClick={() => setSelection({ ...selection, indices: [] })}>
                            取消选择
                        </Button>
                        <label>
                            精确位移（米）
                            <div className="previs-modeling__axes">
                                {axis.map((value, i) => (
                                    <InputNumber key={i} aria-label={`网格位移 ${"XYZ"[i]}`} value={value} step={0.1} onChange={(n) => setAxis(axis.map((old, j) => (j === i ? Number(n ?? 0) : old)) as typeof axis)} />
                                ))}
                            </div>
                        </label>
                        <Button block disabled={!selectedVertices.length} onClick={() => operate(() => transformPrevisMesh(mesh, selectedVertices, new Matrix4().makeTranslation(...axis)), false)}>
                            应用位移
                        </Button>
                        <label>
                            挤出距离（米）
                            <InputNumber value={distance} step={0.1} onChange={(v) => setDistance(v ?? 0.2)} />
                        </label>
                        <Button block disabled={selection.mode !== "face" || !selection.indices.length} onClick={() => operate(() => extrudePrevisMeshFaces(mesh, selection.indices, distance))}>
                            挤出选中面
                        </Button>
                        <Button block disabled={selection.mode !== "face" || !selection.indices.length} onClick={() => operate(() => deletePrevisMeshFaces(mesh, selection.indices))}>
                            删除选中面
                        </Button>
                        <Button block disabled={selection.mode !== "edge" || !selection.indices.length} onClick={() => operate(() => splitPrevisMeshEdges(mesh, selection.indices))}>
                            拆分选中边
                        </Button>
                        <Button block disabled={selection.mode !== "vertex" || selection.indices.length < 2} onClick={() => operate(() => mergePrevisMeshVertices(mesh, selection.indices))}>
                            合并顶点到中心
                        </Button>
                        <p className="previs-modeling__note">几何变化后自动重算法线</p>
                        <label>
                            基础形状
                            <Segmented
                                options={[
                                    { label: "方块", value: "box" },
                                    { label: "球", value: "sphere" },
                                    { label: "圆柱", value: "cylinder" },
                                ]}
                                value=""
                                onChange={(value) => operate(() => createPrevisMeshSeed(value as "box" | "sphere" | "cylinder"))}
                            />
                        </label>
                        <p className="previs-modeling__note">编辑副本保留几何与面颜色。纹理、骨架和动画不进入本编辑器；源对象保留。</p>
                    </aside>
                </div>
                <footer className="previs-modeling__footer">
                    <span>保存为独立模型素材，支持再次编辑</span>
                    <Button disabled={saving} onClick={onClose}>
                        取消
                    </Button>
                    <Button type="primary" loading={saving} disabled={!name.trim()} onClick={() => void save()}>
                        {replacing ? "保存网格修改" : "加入场景"}
                    </Button>
                </footer>
            </div>
        </AppModal>
    );
}

function MeshCanvas({
    mesh,
    initial,
    selection,
    tool,
    onPick,
    onPreview,
    onCommit,
    onError,
    onCamera,
}: {
    mesh: PrevisEditableMesh;
    initial: PrevisEditableMesh;
    selection: PrevisMeshSelection;
    tool: "translate" | "rotate" | "scale";
    onPick: (index: number, additive: boolean) => void;
    onPreview: (mesh: PrevisEditableMesh) => void;
    onCommit: (mesh: PrevisEditableMesh, base: PrevisEditableMesh) => void;
    onError: (message: string) => void;
    onCamera: (camera: Camera) => void;
}) {
    const { camera, invalidate, raycaster } = useThree();
    const pivot = useMemo(() => new Object3D(), []),
        dragging = useRef(false);
    const [orbitEnabled, setOrbitEnabled] = useState(true);
    const gesture = useRef<{ base: PrevisEditableMesh; latest: PrevisEditableMesh; inverse: Matrix4; indices: number[] } | null>(null);
    const bounds = useMemo(() => previsMeshBounds(initial), [initial]);
    const target = useMemo(() => new Vector3(...bounds.min).add(new Vector3(...bounds.max)).multiplyScalar(0.5), [bounds]);
    const span = Math.max(...bounds.max.map((v, i) => v - bounds.min[i]), 0.1);
    useEffect(() => {
        camera.position.copy(target).add(new Vector3(1.4, 1, 1.8).multiplyScalar(span));
        camera.near = span / 1000;
        camera.far = Math.max(100, span * 100);
        camera.lookAt(target);
        camera.updateProjectionMatrix();
        onCamera(camera);
        invalidate();
    }, [camera, initial, invalidate, span, target]);
    useEffect(() => {
        raycaster.params.Points.threshold = span * 0.015;
        raycaster.params.Line.threshold = span * 0.01;
    }, [raycaster, span]);
    const geometry = useMemo(() => buildPrevisMeshGeometry(mesh, selection.mode === "face" ? selection.indices : []), [mesh, selection]);
    const edges = useMemo(() => previsMeshEdges(mesh), [mesh]);
    const overlays = useMemo(() => {
        const points = new BufferGeometry(),
            lines = new BufferGeometry(),
            selected = new Set(selection.indices);
        const positions = mesh.vertices.flat(),
            pointColors = mesh.vertices.flatMap((_, i) => {
                const c = new Color(selection.mode === "vertex" && selected.has(i) ? "#e2810a" : "#334155");
                return [c.r, c.g, c.b];
            });
        points.setAttribute("position", new Float32BufferAttribute(positions, 3));
        points.setAttribute("color", new Float32BufferAttribute(pointColors, 3));
        lines.setAttribute(
            "position",
            new Float32BufferAttribute(
                edges.flatMap(([a, b]) => [...mesh.vertices[a], ...mesh.vertices[b]]),
                3,
            ),
        );
        lines.setAttribute(
            "color",
            new Float32BufferAttribute(
                edges.flatMap((_, i) => {
                    const c = new Color(selection.mode === "edge" && selected.has(i) ? "#ed8508" : "#526374");
                    return [c.r, c.g, c.b, c.r, c.g, c.b];
                }),
                3,
            ),
        );
        return { points, lines };
    }, [edges, mesh, selection]);
    useEffect(
        () => () => {
            geometry.dispose();
            overlays.points.dispose();
            overlays.lines.dispose();
        },
        [geometry, overlays],
    );
    useEffect(() => {
        if (!dragging.current) {
            pivot.position.copy(previsMeshSelectionCenter(mesh, selection));
            pivot.rotation.set(0, 0, 0);
            pivot.scale.set(1, 1, 1);
            pivot.updateMatrixWorld(true);
            invalidate();
        }
    }, [mesh, selection, pivot, invalidate]);
    const finishRef = useRef(() => {});
    finishRef.current = () => {
        const active = gesture.current;
        gesture.current = null;
        dragging.current = false;
        setOrbitEnabled(true);
        if (active) {
            onCommit(active.latest, active.base);
            pivot.position.copy(previsMeshSelectionCenter(active.latest, selection));
            pivot.rotation.set(0, 0, 0);
            pivot.scale.set(1, 1, 1);
            pivot.updateMatrixWorld(true);
            invalidate();
        }
    };
    useEffect(() => {
        const finish = () => finishRef.current();
        const hide = () => {
            if (document.hidden) finish();
        };
        window.addEventListener("blur", finish);
        window.addEventListener("pointercancel", finish);
        document.addEventListener("visibilitychange", hide);
        return () => {
            window.removeEventListener("blur", finish);
            window.removeEventListener("pointercancel", finish);
            document.removeEventListener("visibilitychange", hide);
        };
    }, []);
    const click = (event: ThreeEvent<MouseEvent>, mode: PrevisMeshSelection["mode"], index?: number) => {
        if (dragging.current || event.delta > 4 || selection.mode !== mode || index === undefined) return;
        event.stopPropagation();
        onPick(index, event.shiftKey || event.ctrlKey || event.metaKey);
    };
    return (
        <>
            <mesh geometry={geometry} onClick={(e) => click(e, "face", e.faceIndex ?? undefined)}>
                <meshStandardMaterial vertexColors side={DoubleSide} polygonOffset polygonOffsetFactor={1} polygonOffsetUnits={1} />
            </mesh>
            <lineSegments geometry={overlays.lines} onClick={(e) => click(e, "edge", e.index === undefined ? undefined : Math.floor(e.index / 2))}>
                <lineBasicMaterial vertexColors depthTest={selection.mode !== "edge"} />
            </lineSegments>
            {selection.mode === "vertex" ? (
                <points geometry={overlays.points} onClick={(e) => click(e, "vertex", e.index)}>
                    <pointsMaterial vertexColors size={7} sizeAttenuation={false} depthTest={false} />
                </points>
            ) : null}
            <OrbitControls makeDefault target={target} enabled={orbitEnabled} />
            <primitive object={pivot} dispose={null} />
            {selection.indices.length ? (
                <TransformControls
                    object={pivot}
                    mode={tool}
                    size={0.8}
                    onMouseDown={() => {
                        dragging.current = true;
                        setOrbitEnabled(false);
                        pivot.updateMatrixWorld(true);
                        gesture.current = { base: mesh, latest: mesh, inverse: pivot.matrixWorld.clone().invert(), indices: previsMeshSelectedVertices(mesh, selection) };
                    }}
                    onObjectChange={() => {
                        const active = gesture.current;
                        if (!active) return;
                        try {
                            pivot.updateMatrixWorld(true);
                            const next = transformPrevisMesh(active.base, active.indices, pivot.matrixWorld.clone().multiply(active.inverse));
                            active.latest = next;
                            onPreview(next);
                            onError("");
                        } catch (reason) {
                            onError(reason instanceof Error ? reason.message : "网格变换无效");
                        }
                    }}
                    onMouseUp={() => finishRef.current()}
                />
            ) : null}
        </>
    );
}
