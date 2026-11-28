import { describe, expect, test } from "bun:test";
import { Box3 } from "three";
import cases from "../../backend/internal/app/testdata/previs-sculpt-cases.json";
import { applyPrevisSculptShadows, buildPrevisSculptGroup, createPrevisSculptObject, disposePrevisSculptGroup, normalizePrevisSculptRecipe, patchPrevisSculptRecipe, PrevisSculptError } from "../src/lib/canvas/previs/previs-sculpt";

describe("Sculpt 与 Go 共用的配方合同", () => {
    for (const fixture of cases)
        test(fixture.name, () => {
            const result = normalizePrevisSculptRecipe(fixture.input);
            expect(result.ok).toBe(fixture.valid);
            if (result.ok && fixture.valid) {
                expect(result.recipe).toEqual("normalized" in fixture ? fixture.normalized : undefined);
                expect(normalizePrevisSculptRecipe(result.recipe)).toEqual(result);
                const group = buildPrevisSculptGroup(result.recipe);
                try {
                    expect(new Box3().setFromObject(group).isEmpty()).toBe(false);
                } finally {
                    disposePrevisSculptGroup(group);
                }
            }
        });
    test("补丁与真实几何范围", () => {
        const result = normalizePrevisSculptRecipe({ parts: [{ id: "body", shape: "box", size: [2, 1, 1], position: [0, 0.5, 0] }] });
        if (!result.ok) throw new Error(result.error);
        const object = createPrevisSculptObject(result.recipe, "箱体");
        expect(object.kind).toBe("sculpt");
        expect(object.modelBounds).toEqual({ min: [-1, 0, -0.5], max: [1, 1, 0.5] });
        const patch = patchPrevisSculptRecipe(result.recipe, [{ set: "body", color: "#ff8800" }]);
        expect(patch.ok).toBe(true);
        expect(result.recipe.parts[0].color).toBe("#c2c6c8");
    });
    test("同一造型重复清理只释放一次几何", () => {
        const group = buildPrevisSculptGroup({ version: 1, parts: [{ id: "body", shape: "box", size: [1, 1, 1] }] });
        let disposed = 0;
        group.traverse((node) => {
            if ("geometry" in node) (node as import("three").Mesh).geometry.addEventListener("dispose", () => disposed++);
        });
        disposePrevisSculptGroup(group);
        disposePrevisSculptGroup(group);
        expect(disposed).toBe(1);
    });
    test("对象阴影开关保留玻璃透光语义", () => {
        const group = buildPrevisSculptGroup({
            version: 1,
            parts: [
                { id: "glass", shape: "box", size: [1, 1, 1], material: "glass" },
                { id: "clay", shape: "box", size: [1, 1, 1], position: [2, 0, 0] },
            ],
        });
        try {
            applyPrevisSculptShadows(group, false, false);
            expect(group.getObjectByName("clay")!.castShadow).toBe(false);
            applyPrevisSculptShadows(group, true, true);
            expect(group.getObjectByName("clay")!.castShadow).toBe(true);
            expect(group.getObjectByName("glass")!.castShadow).toBe(false);
        } finally {
            disposePrevisSculptGroup(group);
        }
    });
    test("切割失败提供结构化诊断且不覆盖全局日志", () => {
        const warn = console.warn;
        expect(() =>
            buildPrevisSculptGroup({
                version: 1,
                parts: [
                    { id: "body", shape: "box", size: [1, 1, 1] },
                    { id: "cut", shape: "box", size: [2, 2, 2], parent: "body", op: "subtract" },
                ],
            }),
        ).toThrow(PrevisSculptError);
        expect(console.warn).toBe(warn);
    });
});
