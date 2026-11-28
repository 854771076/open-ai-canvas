import { BoxGeometry, BufferGeometry, Color, CylinderGeometry, Group, InstancedMesh, Matrix4, Mesh, MeshStandardMaterial, Object3D, SphereGeometry, Vector3 } from "three";
import { buildPrevisMeshGeometry, PREVIS_MESH_LIMITS, validatePrevisMesh, type PrevisEditableMesh } from "./previs-mesh";
import type { PrevisVec3 } from "@/types/previs";

/** 克隆生成结果的顶点，不持有目录缓存或导入模型的 GPU 资源。相同坐标焊成公共顶点供拓扑操作。 */
export function previsObjectToEditableMesh(root: Object3D): PrevisEditableMesh {
    const vertices: PrevisVec3[] = [],
        triangles: PrevisEditableMesh["triangles"] = [],
        colors: string[] = [];
    const lookup = new Map<string, number>();
    root.updateMatrixWorld(true);
    root.traverseVisible((node) => {
        if (!(node instanceof Mesh)) return;
        if ("isSkinnedMesh" in node || Object.keys(node.geometry.morphAttributes).length) throw new Error("带骨架或变形动画的模型需先烘焙为静态模型");
        const geometry = node.geometry as BufferGeometry,
            positions = geometry.getAttribute("position"),
            index = geometry.getIndex(),
            vertexColors = geometry.getAttribute("color");
        if (!positions) return;
        const instances = node instanceof InstancedMesh ? node.count : 1;
        const count = index?.count ?? positions.count;
        const start = Math.max(0, geometry.drawRange.start);
        const end = Math.min(count, start + geometry.drawRange.count);
        // CSG 的缓冲容量可能大于实际绘制范围，不能把尾部预分配空间当作真实三角面。
        if (triangles.length + Math.floor((end - start) / 3) * instances > PREVIS_MESH_LIMITS.triangles) throw new Error("转换超过 4 万三角面，请先减少细分或实例数量");
        for (let instance = 0; instance < instances; instance++) {
            const matrix = node.matrixWorld.clone();
            if (node instanceof InstancedMesh) {
                const local = new Matrix4();
                node.getMatrixAt(instance, local);
                matrix.multiply(local);
            }
            for (let i = start; i + 2 < end; i += 3) {
                const group = geometry.groups.find((g) => i >= g.start && i < g.start + g.count);
                const material = (Array.isArray(node.material) ? node.material[group?.materialIndex ?? 0] : node.material) as MeshStandardMaterial;
                const base = material?.color ?? new Color("#c2c6c8"),
                    color = base.clone();
                const t = [0, 1, 2].map((offset) => {
                    const n = index ? index.getX(i + offset) : i + offset;
                    const v = new Vector3().fromBufferAttribute(positions, n).applyMatrix4(matrix);
                    const key = [v.x, v.y, v.z].map((x) => Math.round(x * 1e7)).join(":");
                    let id = lookup.get(key);
                    if (id === undefined) {
                        if (vertices.length >= PREVIS_MESH_LIMITS.vertices) throw new Error("转换超过 2 万顶点，请先减少模型细分");
                        id = vertices.length;
                        vertices.push([v.x, v.y, v.z]);
                        lookup.set(key, id);
                    }
                    if (vertexColors && material?.vertexColors && offset === 0) color.multiply(new Color().fromBufferAttribute(vertexColors, n));
                    return id;
                }) as [number, number, number];
                // 负缩放会反转绕序；烘焙到顶点后必须翻面，才能继续使用正面法线与挤出方向。
                if (matrix.determinant() < 0) [t[1], t[2]] = [t[2], t[1]];
                triangles.push(t);
                colors.push(`#${color.getHexString()}`);
            }
        }
    });
    return validatePrevisMesh({ version: 1, vertices, triangles, colors });
}

export function createPrevisMeshSeed(shape: "box" | "sphere" | "cylinder" = "box"): PrevisEditableMesh {
    const geometry = shape === "sphere" ? new SphereGeometry(0.5, 16, 12) : shape === "cylinder" ? new CylinderGeometry(0.5, 0.5, 1, 16) : new BoxGeometry(1, 1, 1);
    geometry.translate(0, 0.5, 0);
    const material = new MeshStandardMaterial({ color: "#b7c4ce" });
    try {
        return previsObjectToEditableMesh(new Mesh(geometry, material));
    } finally {
        geometry.dispose();
        material.dispose();
    }
}

export async function exportPrevisMeshGLB(input: PrevisEditableMesh): Promise<Blob> {
    validatePrevisMesh(input);
    const { GLTFExporter } = await import("three/examples/jsm/exporters/GLTFExporter.js");
    const geometry = buildPrevisMeshGeometry(input),
        material = new MeshStandardMaterial({ vertexColors: true, roughness: 0.78 });
    const root = new Group();
    root.add(new Mesh(geometry, material));
    try {
        const result = await new GLTFExporter().parseAsync(root, { binary: true });
        if (!(result instanceof ArrayBuffer)) throw new Error("网格 GLB 导出失败");
        return new Blob([result], { type: "model/gltf-binary" });
    } finally {
        geometry.dispose();
        material.dispose();
    }
}
