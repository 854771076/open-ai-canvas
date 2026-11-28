import { expect, test } from "bun:test";
import { loadPrevisMeshSource } from "../src/services/previs-mesh-source";
import { createPrevisCatalogProp, createPrevisObject } from "../src/lib/canvas/previs/previs-scene";
import { buildPrevisPropParts } from "../src/lib/canvas/previs/previs-prop-geometry";
import { previsMeshBounds } from "../src/lib/canvas/previs/previs-mesh";

test("平面复制保留视口的 XY 朝向与 1.6×1 米尺寸", async () => {
    const mesh = await loadPrevisMeshSource(createPrevisObject("plane", "平面"), new AbortController().signal);
    const bounds = previsMeshBounds(mesh);
    expect(mesh.triangles).toHaveLength(2);
    expect(bounds.max[0] - bounds.min[0]).toBeCloseTo(1.6);
    expect(bounds.max[1] - bounds.min[1]).toBeCloseTo(1);
    expect(bounds.max[2] - bounds.min[2]).toBe(0);
});

test("目录道具复制不改写或释放其他实例共用的几何", async () => {
    const propId = "shotloom-shape-box",
        params = { width: 2, height: 3, depth: 4 };
    const parts = buildPrevisPropParts(propId, 0, params);
    const original = Array.from(parts[0].geometry.getAttribute("position").array);
    let disposed = 0;
    const listener = () => disposed++;
    parts[0].geometry.addEventListener("dispose", listener);
    try {
        const object = createPrevisCatalogProp({ propId, name: "方块", color: "#ff0000", params });
        const mesh = await loadPrevisMeshSource(object, new AbortController().signal);
        expect(previsMeshBounds(mesh).max[1]).toBeCloseTo(3);
        expect(Array.from(parts[0].geometry.getAttribute("position").array)).toEqual(original);
        expect(disposed).toBe(0);
    } finally {
        parts[0].geometry.removeEventListener("dispose", listener);
    }
});
