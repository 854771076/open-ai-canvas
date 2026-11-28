import { Box3, BufferGeometry, Color, Float32BufferAttribute, Group, Matrix4, Mesh, MeshStandardMaterial, Vector3 } from "three";
import type { PrevisMeshData, PrevisObject, PrevisScene, PrevisVec3 } from "@/types/previs";

/** 导入模型导出为 GLB；Agent 创建的网格保留有界拓扑，便于下一轮继续编辑。 */
export type PrevisEditableMesh = PrevisMeshData;
export type PrevisMeshSelection = { mode: "vertex" | "edge" | "face"; indices: number[] };
export const PREVIS_MESH_LIMITS = { vertices: 20_000, triangles: 40_000, extent: 10_000 } as const;
const PREVIS_SCENE_BYTES_LIMIT = 1 << 20;

const point = (p: PrevisVec3) => new Vector3(...p);
const tuple = (p: Vector3): PrevisVec3 => [p.x, p.y, p.z];
const edgeKey = (a: number, b: number) => (a < b ? `${a}:${b}` : `${b}:${a}`);

export function validatePrevisMesh(input: PrevisEditableMesh): PrevisEditableMesh {
    if (!input || input.version !== 1 || !Array.isArray(input.vertices) || !Array.isArray(input.triangles) || !Array.isArray(input.colors)) throw new Error("网格数据格式无效");
    if (Object.keys(input).some((key) => !["version", "vertices", "triangles", "colors"].includes(key))) throw new Error("网格包含未知字段");
    if (input.vertices.length < 3 || input.vertices.length > PREVIS_MESH_LIMITS.vertices || !input.triangles.length || input.triangles.length > PREVIS_MESH_LIMITS.triangles) throw new Error("网格需包含三角面，且不超过 2 万顶点、4 万三角面");
    input.vertices.forEach((v) => {
        if (!Array.isArray(v) || v.length !== 3 || v.some((n) => !Number.isFinite(n) || Math.abs(n) > PREVIS_MESH_LIMITS.extent)) throw new Error("网格顶点坐标无效或超出编辑范围");
    });
    if (input.colors.length !== input.triangles.length || input.colors.some((c) => !/^#[0-9a-f]{6}$/i.test(c))) throw new Error("网格面颜色无效");
    input.triangles.forEach((t) => {
        if (!Array.isArray(t) || t.length !== 3 || t.some((i) => !Number.isInteger(i) || i < 0 || i >= input.vertices.length) || new Set(t).size !== 3) throw new Error("三角面顶点索引无效");
        const [a, b, c] = t.map((i) => point(input.vertices[i]));
        if (b.sub(a).cross(c.sub(a)).lengthSq() < 1e-16) throw new Error("操作会产生零面积三角面，请调整顶点位置");
    });
    return input;
}

export function previsMeshEdges(mesh: PrevisEditableMesh): Array<[number, number]> {
    const found = new Map<string, [number, number]>();
    mesh.triangles.forEach((t) =>
        t.forEach((a, i) => {
            const b = t[(i + 1) % 3];
            if (!found.has(edgeKey(a, b))) found.set(edgeKey(a, b), [Math.min(a, b), Math.max(a, b)]);
        }),
    );
    return [...found.values()];
}

export function previsMeshSelectedVertices(mesh: PrevisEditableMesh, selection: PrevisMeshSelection): number[] {
    const found = new Set<number>();
    const edges = selection.mode === "edge" ? previsMeshEdges(mesh) : [];
    selection.indices.forEach((i) => {
        if (selection.mode === "vertex") {
            if (mesh.vertices[i]) found.add(i);
        } else (selection.mode === "edge" ? edges[i] : mesh.triangles[i])?.forEach((v) => found.add(v));
    });
    return [...found];
}

export function previsMeshSelectionCenter(mesh: PrevisEditableMesh, selection: PrevisMeshSelection): Vector3 {
    const indices = previsMeshSelectedVertices(mesh, selection);
    return indices.reduce((center, i) => center.add(point(mesh.vertices[i])), new Vector3()).divideScalar(indices.length || 1);
}

export function transformPrevisMesh(mesh: PrevisEditableMesh, indices: readonly number[], matrix: Matrix4): PrevisEditableMesh {
    const selected = new Set(indices);
    return validatePrevisMesh({ ...mesh, vertices: mesh.vertices.map((v, i) => (selected.has(i) ? tuple(point(v).applyMatrix4(matrix)) : v)) });
}

function compactMesh(mesh: PrevisEditableMesh): PrevisEditableMesh {
    const used = new Set(mesh.triangles.flat());
    const indices = new Map<number, number>();
    const vertices = mesh.vertices.filter((_, i) => {
        if (!used.has(i)) return false;
        indices.set(i, indices.size);
        return true;
    });
    return validatePrevisMesh({ ...mesh, vertices, triangles: mesh.triangles.map((t) => t.map((i) => indices.get(i)!) as [number, number, number]) });
}

export function deletePrevisMeshFaces(mesh: PrevisEditableMesh, faces: readonly number[]): PrevisEditableMesh {
    const selected = new Set(faces);
    return compactMesh({ ...mesh, triangles: mesh.triangles.filter((_, i) => !selected.has(i)), colors: mesh.colors.filter((_, i) => !selected.has(i)) });
}

/** 区域挤出只连接边界边；内部公共边不生成重叠侧面。 */
export function extrudePrevisMeshFaces(mesh: PrevisEditableMesh, faces: readonly number[], distance: number): PrevisEditableMesh {
    if (!Number.isFinite(distance) || Math.abs(distance) < 0.001) throw new Error("挤出距离至少为 0.001 米");
    const selected = new Set(faces);
    if (!selected.size || faces.some((i) => !mesh.triangles[i])) throw new Error("请先选择要挤出的面");
    const edges = new Map<string, { a: number; b: number; count: number; color: string }>();
    const allEdges = new Map<string, number>();
    mesh.triangles.forEach((t) =>
        t.forEach((a, i) => {
            const key = edgeKey(a, t[(i + 1) % 3]);
            allEdges.set(key, (allEdges.get(key) ?? 0) + 1);
        }),
    );
    const normal = new Vector3();
    selected.forEach((i) => {
        const t = mesh.triangles[i];
        const [a, b, c] = t.map((v) => point(mesh.vertices[v]));
        normal.add(b.sub(a).cross(c.sub(a)));
        t.forEach((a, j) => {
            const b = t[(j + 1) % 3],
                key = edgeKey(a, b),
                old = edges.get(key);
            if ((allEdges.get(key) ?? 0) > 2 || (old && old.a === a && old.b === b)) throw new Error("选区包含非流形边或反向面，请先修复拓扑");
            edges.set(key, { a, b, count: (old?.count ?? 0) + 1, color: mesh.colors[i] });
        });
    });
    if (normal.lengthSq() < 1e-16) throw new Error("选区没有明确的挤出方向，请选择同向的面");
    normal.normalize().multiplyScalar(distance);
    const vertices = mesh.vertices.map((v) => [...v] as PrevisVec3),
        copies = new Map<number, number>();
    selected.forEach((i) =>
        mesh.triangles[i].forEach((v) => {
            if (copies.has(v)) return;
            copies.set(v, vertices.length);
            vertices.push(tuple(point(mesh.vertices[v]).add(normal)));
        }),
    );
    const triangles = mesh.triangles.filter((_, i) => !selected.has(i));
    const colors = mesh.colors.filter((_, i) => !selected.has(i));
    selected.forEach((i) => {
        triangles.push(mesh.triangles[i].map((v) => copies.get(v)!) as [number, number, number]);
        colors.push(mesh.colors[i]);
    });
    edges.forEach(({ a, b, count, color }) => {
        if (count !== 1) return;
        triangles.push([a, b, copies.get(b)!], [a, copies.get(b)!, copies.get(a)!]);
        colors.push(color, color);
    });
    return compactMesh({ version: 1, vertices, triangles, colors });
}

export function splitPrevisMeshEdges(mesh: PrevisEditableMesh, selection: readonly number[]): PrevisEditableMesh {
    const edges = previsMeshEdges(mesh);
    const splits = new Map<string, number>();
    selection.forEach((i) => {
        const edge = edges[i];
        if (!edge) throw new Error("选择的边已失效");
        const key = edgeKey(...edge);
        if (splits.has(key)) return;
        splits.set(key, mesh.vertices.length + splits.size);
    });
    // 一条边在每个相邻面中增加一个三角面；先检查容量和非流形边，避免超限输入做完细分才失败。
    const incidents = new Map<string, number>();
    let triangleCount = mesh.triangles.length;
    mesh.triangles.forEach((t) => {
        t.forEach((a, i) => {
            const key = edgeKey(a, t[(i + 1) % 3]);
            if (!splits.has(key)) return;
            const count = (incidents.get(key) ?? 0) + 1;
            if (count > 2) throw new Error("不支持拆分非流形边");
            incidents.set(key, count);
            triangleCount++;
        });
    });
    if (mesh.vertices.length + splits.size > PREVIS_MESH_LIMITS.vertices || triangleCount > PREVIS_MESH_LIMITS.triangles) throw new Error("拆分后超过 2 万顶点、4 万三角面的编辑上限");
    const vertices = [...mesh.vertices];
    splits.forEach((_, key) => {
        const [a, b] = key.split(":").map(Number);
        vertices.push(tuple(point(vertices[a]).add(point(vertices[b])).multiplyScalar(0.5)));
    });
    const triangles: PrevisEditableMesh["triangles"] = [],
        colors: string[] = [];
    mesh.triangles.forEach((t, face) => {
        // 每个原始面只处理自己的三条边，局部最多四个碎片，不随全网格的选中边数反复扫描。
        let fragments = [t];
        t.forEach((a, i) => {
            const b = t[(i + 1) % 3],
                middle = splits.get(edgeKey(a, b));
            if (middle === undefined) return;
            fragments = fragments.flatMap((fragment) => {
                const index = fragment.findIndex((v, j) => (v === a && fragment[(j + 1) % 3] === b) || (v === b && fragment[(j + 1) % 3] === a));
                if (index === -1) return [fragment];
                const start = fragment[index],
                    end = fragment[(index + 1) % 3],
                    other = fragment[(index + 2) % 3];
                return [
                    [start, middle, other],
                    [middle, end, other],
                ] as PrevisEditableMesh["triangles"];
            });
        });
        triangles.push(...fragments);
        fragments.forEach(() => colors.push(mesh.colors[face]));
    });
    return validatePrevisMesh({ version: 1, vertices, triangles, colors });
}

export function mergePrevisMeshVertices(mesh: PrevisEditableMesh, indices: readonly number[]): PrevisEditableMesh {
    const selected = new Set(indices);
    if (selected.size < 2 || indices.some((i) => !mesh.vertices[i])) throw new Error("请至少选择两个顶点");
    const target = Math.min(...indices),
        center = previsMeshSelectionCenter(mesh, { mode: "vertex", indices: [...selected] });
    const vertices = mesh.vertices.map((v, i) => (i === target ? tuple(center) : v));
    const triangles: PrevisEditableMesh["triangles"] = [],
        colors: string[] = [],
        seen = new Set<string>();
    mesh.triangles.forEach((t, i) => {
        const next = t.map((v) => (selected.has(v) ? target : v)) as [number, number, number];
        const key = [...next].sort((a, b) => a - b).join(":");
        if (new Set(next).size < 3 || seen.has(key)) return;
        triangles.push(next);
        colors.push(mesh.colors[i]);
        seen.add(key);
    });
    return compactMesh({ version: 1, vertices, triangles, colors });
}

export function buildPrevisMeshGeometry(mesh: PrevisEditableMesh, selectedFaces: readonly number[] = []): BufferGeometry {
    const positions: number[] = [],
        colors: number[] = [],
        selected = new Set(selectedFaces);
    mesh.triangles.forEach((t, i) => {
        const color = new Color(selected.has(i) ? "#ffb020" : mesh.colors[i]);
        t.forEach((v) => {
            positions.push(...mesh.vertices[v]);
            colors.push(color.r, color.g, color.b);
        });
    });
    const geometry = new BufferGeometry();
    geometry.setAttribute("position", new Float32BufferAttribute(positions, 3));
    geometry.setAttribute("color", new Float32BufferAttribute(colors, 3));
    geometry.computeVertexNormals();
    geometry.computeBoundingBox();
    geometry.computeBoundingSphere();
    return geometry;
}

export function previsMeshBounds(mesh: PrevisEditableMesh): { min: PrevisVec3; max: PrevisVec3 } {
    const box = new Box3().setFromPoints(mesh.vertices.map(point));
    return { min: tuple(box.min), max: tuple(box.max) };
}

/** 数据源保持为拓扑；派生几何沿用视口的材质模式与资源释放流程。 */
export function buildPrevisMeshGroup(input: PrevisEditableMesh): Group {
    const mesh = validatePrevisMesh(input);
    const group = new Group();
    group.add(new Mesh(buildPrevisMeshGeometry(mesh), new MeshStandardMaterial({ vertexColors: true, roughness: 0.78 })));
    return group;
}

export function replacePrevisMeshTopology(source: PrevisObject, mesh: PrevisEditableMesh, name: string): PrevisObject {
    if (source.kind !== "mesh") throw new Error("只能替换三角网格对象的拓扑");
    validatePrevisMesh(mesh);
    // 保留对象 ID、关键帧、变换及父子关系，拓扑不与编辑会话共享可变数组。
    return { ...source, name, mesh: structuredClone(mesh), modelBounds: previsMeshBounds(mesh) };
}

/** 与服务端场景总量上限对齐，至少在上传 GLB 前拒绝单个内联场景已经超限的修改。 */
export function assertPrevisMeshSceneSize(scene: PrevisScene): void {
    if (new TextEncoder().encode(JSON.stringify([scene])).byteLength > PREVIS_SCENE_BYTES_LIMIT) throw new Error("场景数据超过 1 MiB，请减少网格细分后保存");
}
