import { describe, expect, test } from "bun:test";
import { Matrix4, Vector3 } from "three";
import {
    deletePrevisMeshFaces,
    extrudePrevisMeshFaces,
    mergePrevisMeshVertices,
    previsMeshEdges,
    previsMeshSelectedVertices,
    PREVIS_MESH_LIMITS,
    splitPrevisMeshEdges,
    transformPrevisMesh,
    validatePrevisMesh,
    type PrevisEditableMesh,
} from "../src/lib/canvas/previs/previs-mesh";

const square = (): PrevisEditableMesh => ({
    version: 1,
    vertices: [
        [0, 0, 0],
        [1, 0, 0],
        [1, 1, 0],
        [0, 1, 0],
    ],
    triangles: [
        [0, 1, 2],
        [0, 2, 3],
    ],
    colors: ["#ffffff", "#aabbcc"],
});

describe("可编辑三角网格", () => {
    test("拒绝非法索引、坐标、退化面与空网格", () => {
        expect(() => validatePrevisMesh({ ...square(), triangles: [[0, 1, 4]], colors: ["#ffffff"] })).toThrow("索引");
        expect(() =>
            validatePrevisMesh({
                ...square(),
                vertices: [
                    [NaN, 0, 0],
                    [1, 0, 0],
                    [0, 1, 0],
                ],
            }),
        ).toThrow("坐标");
        expect(() => validatePrevisMesh({ ...square(), triangles: [[0, 0, 1]], colors: ["#ffffff"] })).toThrow("索引");
        expect(() => deletePrevisMeshFaces(square(), [0, 1])).toThrow("三角面");
    });
    test("共享边只出现一次，面选区转换为去重顶点", () => {
        expect(previsMeshEdges(square())).toHaveLength(5);
        expect(previsMeshSelectedVertices(square(), { mode: "face", indices: [0, 1] })).toEqual([0, 1, 2, 3]);
    });
    test("修改独立副本且移动后仍验证退化", () => {
        const original = square();
        const next = transformPrevisMesh(original, [0], new Matrix4().makeTranslation(0, 0, 1));
        expect(next.vertices[0]).toEqual([0, 0, 1]);
        expect(original.vertices[0]).toEqual([0, 0, 0]);
        expect(() => transformPrevisMesh(original, [0], new Matrix4().makeTranslation(1, 0, 0))).toThrow("面积");
    });
    test("区域挤出不在公共对角线上生成内部侧面", () => {
        const next = extrudePrevisMeshFaces(square(), [0, 1], 1);
        expect(next.vertices).toHaveLength(8);
        expect(next.triangles).toHaveLength(10);
        expect(next.vertices.filter((v) => v[2] === 1)).toHaveLength(4);
        expect(square().vertices).toHaveLength(4);
    });
    test("公共边拆分同时更新相邻两面", () => {
        const mesh = square(),
            edge = previsMeshEdges(mesh).findIndex(([a, b]) => a === 0 && b === 2);
        const next = splitPrevisMeshEdges(mesh, [edge]);
        expect(next.vertices).toHaveLength(5);
        expect(next.triangles).toHaveLength(4);
        expect(next.triangles.every((t) => t.includes(4))).toBe(true);
    });
    test("同一面拆分两条或三条边保留面积、绕序和颜色，共享中点不产生裂缝", () => {
        for (const count of [2, 3]) {
            const mesh = square();
            const next = splitPrevisMeshEdges(
                mesh,
                Array.from({ length: count }, (_, i) => i),
            );
            expect(next.vertices).toHaveLength(mesh.vertices.length + count);
            let area = 0;
            next.triangles.forEach((t, face) => {
                const [a, b, c] = t.map((i) => new Vector3(...next.vertices[i]));
                const normal = b.sub(a).cross(c.sub(a));
                expect(normal.z).toBeGreaterThan(0);
                area += normal.length() / 2;
                expect(next.colors[face]).toBe(face < count + 1 ? "#ffffff" : "#aabbcc");
            });
            expect(area).toBeCloseTo(1);
            const outputEdges = previsMeshEdges(next);
            previsMeshEdges(mesh)
                .slice(0, count)
                .forEach(([a, b], i) => {
                    const middle = mesh.vertices.length + i;
                    expect(outputEdges).not.toContainEqual([a, b]);
                    expect(outputEdges).toContainEqual([a, middle]);
                    expect(outputEdges).toContainEqual([b, middle]);
                });
        }
    });
    test("批量拆边只读取原始面常数次，拒绝非流形和超限结果", () => {
        const mesh = square();
        mesh.vertices.push([0, 0, 1]);
        mesh.triangles.push([0, 2, 4]);
        mesh.colors.push("#ffffff");
        const edge = previsMeshEdges(mesh).findIndex(([a, b]) => a === 0 && b === 2);
        expect(() => splitPrevisMeshEdges(mesh, [edge])).toThrow("非流形");

        const large: PrevisEditableMesh = { version: 1, vertices: [], triangles: [], colors: [] };
        // 不依赖机器耗时：记录原面索引读取量，防止恢复为每条选中边遍历整张网格。
        let reads = 0;
        for (let i = 0; i < 200; i++) {
            large.vertices.push([i * 2, 0, 0], [i * 2 + 1, 0, 0], [i * 2, 1, 0]);
            large.triangles.push(
                new Proxy([i * 3, i * 3 + 1, i * 3 + 2] as [number, number, number], {
                    get(target, key, receiver) {
                        if (typeof key === "string" && /^\d+$/.test(key)) reads++;
                        return Reflect.get(target, key, receiver);
                    },
                }),
            );
            large.colors.push("#ffffff");
        }
        const split = splitPrevisMeshEdges(
            large,
            Array.from({ length: 200 }, (_, i) => i * 3),
        );
        expect(split.triangles).toHaveLength(400);
        expect(reads).toBeLessThanOrEqual(100 * large.triangles.length);

        const atLimit = square();
        atLimit.vertices = Array.from({ length: PREVIS_MESH_LIMITS.vertices }, (_, i) => atLimit.vertices[i % 4]);
        expect(() => splitPrevisMeshEdges(atLimit, [0])).toThrow("上限");
        const faceLimit = square();
        faceLimit.triangles = Array.from({ length: PREVIS_MESH_LIMITS.triangles }, () => [0, 1, 2]);
        faceLimit.colors = faceLimit.triangles.map(() => "#ffffff");
        faceLimit.triangles[0] = [0, 2, 3];
        expect(() => splitPrevisMeshEdges(faceLimit, [2])).toThrow("上限");
    });
    test("删除与合并更新索引并清理不再引用的顶点", () => {
        const deleted = deletePrevisMeshFaces(square(), [0]);
        expect(deleted.vertices).toHaveLength(3);
        expect(deleted.colors).toEqual(["#aabbcc"]);
        const merged = mergePrevisMeshVertices(square(), [0, 1]);
        expect(merged.triangles).toHaveLength(1);
        expect(merged.vertices).toHaveLength(3);
    });
    test("非流形公共边不允许挤出", () => {
        const mesh = square();
        mesh.vertices.push([0, 0, 1]);
        mesh.triangles.push([0, 2, 4]);
        mesh.colors.push("#ffffff");
        expect(() => extrudePrevisMeshFaces(mesh, [0], 1)).toThrow("非流形");
    });
});
