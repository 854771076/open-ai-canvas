import { expect, test } from "bun:test";
import cases from "../../backend/internal/app/testdata/previs-mesh-cases.json";
import { assertPrevisMeshSceneSize, buildPrevisMeshGroup, replacePrevisMeshTopology, validatePrevisMesh, type PrevisEditableMesh } from "../src/lib/canvas/previs/previs-mesh";
import { loadPrevisMeshSource } from "../src/services/previs-mesh-source";
import { createPrevisObject, createPrevisScene } from "../src/lib/canvas/previs/previs-scene";
import { resolvePrevisModelBounds } from "../src/lib/canvas/previs/previs-model-bounds";
import type { PrevisObject } from "../src/types/previs";
import { disposePrevisObject3D } from "../src/lib/canvas/previs/previs-resources";
import { Mesh } from "three";

for (const fixture of cases) {
    test(`Agent 与手动编辑器网格合同一致：${fixture.name}`, () => {
        const validate = () => validatePrevisMesh(fixture.mesh as PrevisEditableMesh);
        if (fixture.valid) expect(validate()).toEqual(fixture.mesh);
        else expect(validate).toThrow();
    });
}

test("Agent 网格可直接打开手动编辑器，保留拓扑、尺寸、面颜色且不共享可变数组", async () => {
    const mesh = structuredClone(cases[0].mesh) as PrevisEditableMesh;
    const object: PrevisObject = { ...createPrevisObject(), kind: "mesh", primitive: undefined, mesh, modelBounds: { min: [0, 0, 0], max: [99, 99, 99] } };
    const editable = await loadPrevisMeshSource(object, new AbortController().signal);
    expect(editable).toEqual(mesh);
    editable.vertices[0][0] = -3;
    expect(mesh.vertices[0][0]).toBe(-1);
    expect(resolvePrevisModelBounds(object)).toEqual({ min: [-1, 0, -1], max: [1, 2, 1] });
});

test("Agent 拓扑生成真实视口几何，手动保存保留对象身份和动画", () => {
    const mesh = structuredClone(cases[0].mesh) as PrevisEditableMesh;
    const object: PrevisObject = { ...createPrevisObject(), kind: "mesh", primitive: undefined, mesh, parentId: "parent" };
    object.keyframes = [{ id: "key-1", time: 1, transform: structuredClone(object.transform) }];
    const root = buildPrevisMeshGroup(mesh);
    try {
        const geometry = (root.children[0] as Mesh).geometry;
        expect(geometry.getAttribute("position").count).toBe(12);
        expect(geometry.getAttribute("color").count).toBe(12);
        expect(geometry.boundingBox!.max.y).toBe(2);
        mesh.vertices[3][1] = 4;
        const edited = replacePrevisMeshTopology(object, mesh, "新尖塔");
        expect(edited.id).toBe(object.id);
        expect(edited.parentId).toBe("parent");
        expect(edited.keyframes).toEqual(object.keyframes);
        expect(edited.transform).toEqual(object.transform);
        expect(edited.modelBounds).toEqual({ min: [-1, 0, -1], max: [1, 4, 1] });
        mesh.vertices[3][1] = 8;
        expect(edited.mesh!.vertices[3][1]).toBe(4);
    } finally {
        disposePrevisObject3D(root);
    }
});

test("手动编辑 Agent 网格后在上传前拒绝超过场景 JSON 容量的拓扑", () => {
    const scene = createPrevisScene();
    const mesh = structuredClone(cases[0].mesh) as PrevisEditableMesh;
    scene.objects = [{ ...createPrevisObject(), kind: "mesh", primitive: undefined, mesh }];
    expect(() => assertPrevisMeshSceneSize(scene)).not.toThrow();
    mesh.vertices.push(...Array.from({ length: 19996 }, () => [0.12345678901234567, 0.23456789012345678, 0.34567890123456789] as [number, number, number]));
    expect(() => assertPrevisMeshSceneSize(scene)).toThrow("1 MiB");
});
