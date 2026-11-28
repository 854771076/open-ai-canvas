import { expect, test } from "bun:test";
import { BoxGeometry, Group, InstancedMesh, Matrix4, Mesh, MeshStandardMaterial } from "three";
import { createPrevisMeshSeed, exportPrevisMeshGLB, previsObjectToEditableMesh } from "../src/lib/canvas/previs/previs-mesh-conversion";
import { previsMeshBounds } from "../src/lib/canvas/previs/previs-mesh";
import { buildPrevisSculptGroup, disposePrevisSculptGroup } from "../src/lib/canvas/previs/previs-sculpt";

test("转换尊重绘制范围，不导入缓冲区内未绘制的面", () => {
    const geometry = new BoxGeometry(),
        material = new MeshStandardMaterial();
    geometry.setDrawRange(0, 6);
    expect(previsObjectToEditableMesh(new Mesh(geometry, material)).triangles).toHaveLength(2);
    geometry.dispose();
    material.dispose();
});

test("真实布尔切割的造型可以继续转为可编辑网格", () => {
    const group = buildPrevisSculptGroup({
        version: 1,
        parts: [
            { id: "body", shape: "box", size: [1, 1, 1] },
            { id: "hole", shape: "cylinder", size: [0.4, 2, 0.4], parent: "body", op: "subtract" },
        ],
    });
    try {
        expect(previsObjectToEditableMesh(group).triangles.length).toBeGreaterThan(12);
    } finally {
        disposePrevisSculptGroup(group);
    }
});

test("几何副本焊接方块接缝并保持米制尺寸", () => {
    const mesh = createPrevisMeshSeed();
    expect(mesh.vertices).toHaveLength(8);
    expect(mesh.triangles).toHaveLength(12);
    expect(previsMeshBounds(mesh)).toEqual({ min: [-0.5, 0, -0.5], max: [0.5, 1, 0.5] });
});
test("转换烘焙层级与实例变换，未修改源资源", () => {
    const root = new Group(),
        geometry = new BoxGeometry(),
        material = new MeshStandardMaterial({ color: "#ff0000" });
    const source = new InstancedMesh(geometry, material, 2);
    source.setMatrixAt(0, new Matrix4().makeTranslation(0, 0, 0));
    source.setMatrixAt(1, new Matrix4().makeTranslation(3, 0, 0));
    root.position.y = 2;
    root.add(source);
    const converted = previsObjectToEditableMesh(root);
    expect(converted.triangles).toHaveLength(24);
    expect(previsMeshBounds(converted)).toEqual({ min: [-0.5, 1.5, -0.5], max: [3.5, 2.5, 0.5] });
    expect(converted.colors.every((color) => color === "#ff0000")).toBe(true);
    expect(geometry.getAttribute("position").count).toBe(24);
    geometry.dispose();
    material.dispose();
    source.dispose();
});
test("球体与圆柱可转换成有效拓扑", () => {
    expect(createPrevisMeshSeed("sphere").triangles.length).toBeGreaterThan(100);
    expect(createPrevisMeshSeed("cylinder").triangles.length).toBeGreaterThan(20);
});
test("负缩放转换修正绕序", () => {
    const geometry = new BoxGeometry(),
        material = new MeshStandardMaterial();
    const source = new Mesh(geometry, material);
    const positive = previsObjectToEditableMesh(source);
    source.scale.x = -1;
    const negative = previsObjectToEditableMesh(source);
    expect(negative.triangles[0]).toEqual([positive.triangles[0][0], positive.triangles[0][2], positive.triangles[0][1]]);
    geometry.dispose();
    material.dispose();
});

test("GLB 导出并重新读取保持米制尺寸、面数量和颜色", async () => {
    const old = Object.getOwnPropertyDescriptor(globalThis, "FileReader");
    class TestFileReader {
        result: ArrayBuffer | null = null;
        onloadend?: () => void;
        readAsArrayBuffer(blob: Blob) {
            void blob.arrayBuffer().then((value) => {
                this.result = value;
                this.onloadend?.();
            });
        }
    }
    Object.defineProperty(globalThis, "FileReader", { configurable: true, value: TestFileReader });
    try {
        const input = createPrevisMeshSeed();
        const blob = await exportPrevisMeshGLB(input);
        const { GLTFLoader } = await import("three-stdlib");
        const result = await new GLTFLoader().parseAsync(await blob.arrayBuffer(), "");
        const output = previsObjectToEditableMesh(result.scene);
        expect(output.triangles).toHaveLength(input.triangles.length);
        expect(output.vertices).toHaveLength(input.vertices.length);
        expect(previsMeshBounds(output)).toEqual(previsMeshBounds(input));
        expect(output.colors).toEqual(input.colors);
        result.scene.traverse((node) => {
            if (node instanceof Mesh) {
                node.geometry.dispose();
                const materials = Array.isArray(node.material) ? node.material : [node.material];
                materials.forEach((m) => m.dispose());
            }
        });
    } finally {
        if (old) Object.defineProperty(globalThis, "FileReader", old);
        else Reflect.deleteProperty(globalThis, "FileReader");
    }
});
