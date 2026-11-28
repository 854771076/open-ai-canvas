import { expect, test } from "bun:test";
import { createPrevisModel } from "../src/lib/canvas/previs/previs-scene";
import { previsObjectFootprint } from "../src/lib/canvas/previs/previs-placement";
import { resolvePrevisObjectBox } from "../src/lib/canvas/previs/previs-frame-bounds";
import { dropPrevisObjectToGround } from "../src/lib/canvas/previs/previs-grounding";
import { normalizeSculptRecipe, sculptBounds } from "../src/lib/canvas/previs/sculpt/sculpt-recipe.js";
import { resolvePrevisModelBounds } from "../src/lib/canvas/previs/previs-model-bounds";
import fixtures from "../../backend/internal/app/testdata/previs-sculpt-bounds.json";

test("配方包围盒与 Go 共用镜像、父子旋转和切割排除用例", () => {
    for (const fixture of fixtures) {
        const result = normalizeSculptRecipe(fixture.input);
        if (!result.ok) throw new Error(result.error);
        expect(sculptBounds(result.recipe)).toEqual(fixture.bounds);
    }
});

test("Agent 更新配方后忽略旧派生包围盒", () => {
    expect(resolvePrevisModelBounds({ kind: "sculpt", sculpt: { version: 1, parts: [{ id: "body", shape: "box", size: [4, 2, 2] }] }, modelBounds: { min: [0, 0, 0], max: [1, 1, 1] } })).toEqual({ min: [-2, -1, -1], max: [2, 1, 1] });
});

test("编辑模型的旋转、负缩放与取景占地贴地使用同一包围盒", () => {
    const object = createPrevisModel({ name: "测试网格", url: "blob:mesh" });
    object.meshEditable = true;
    object.modelBounds = { min: [-1, 0, -3], max: [1, 2, 3] };
    object.transform = { position: [0, 0, 0], rotation: [0, 0, Math.PI / 2], scale: [-2, 1, 1] };
    const box = resolvePrevisObjectBox(object);
    expect(box.min.y).toBeCloseTo(-2);
    expect(box.max.y).toBeCloseTo(2);
    const footprint = previsObjectFootprint(object);
    expect(footprint.width).toBeCloseTo(2);
    expect(footprint.depth).toBeCloseTo(6);
    expect(dropPrevisObjectToGround(object).transform.position[1]).toBeCloseTo(2);
});
