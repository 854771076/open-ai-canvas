import { describe, expect, test } from "bun:test";
import { PREVIS_PROP_CATALOG, resolvePrevisPropSize } from "../src/lib/canvas/previs/previs-prop-catalog";
import { buildPrevisPropParts, hasPrevisPropBuilder, previsPropMaterial, previsPropPartsBounds } from "../src/lib/canvas/previs/previs-prop-geometry";
import { DoubleSide, Vector3 } from "three";
import { readFileSync } from "node:fs";
import { resolveShotloomPropAnchors } from "../src/lib/canvas/previs/previs-shotloom";
import { SHOTLOOM_STATIC_ASSETS } from "../src/lib/canvas/previs/shotloom/asset-catalog";
import { box } from "../src/lib/canvas/previs/shotloom/assets/geometry";
import { Group, MeshStandardMaterial } from "three";

describe("Shotloom 静态道具接入", () => {
    test("直角构件保持尺寸，顶点数量不包含无效圆角细分", () => {
        const group = new Group(),
            material = new MeshStandardMaterial();
        const plain = box(group, material, 2, 3, 4, 0, 0, 0, 0);
        const rounded = box(group, material, 2, 3, 4, 0, 0, 0, 0.1);
        try {
            expect(plain.geometry.getAttribute("position").count).toBe(24);
            expect(rounded.geometry.getAttribute("position").count).toBeGreaterThan(24);
            plain.geometry.computeBoundingBox();
            expect(plain.geometry.boundingBox!.getSize(new Vector3()).toArray()).toEqual([2, 3, 4]);
        } finally {
            plain.geometry.dispose();
            rounded.geometry.dispose();
            material.dispose();
        }
    });
    test("136 个现代预设均使用真实构建器且实际尺寸与目录一致", () => {
        const props = PREVIS_PROP_CATALOG.filter((prop) => prop.id.startsWith("shotloom-"));
        expect(props).toHaveLength(136);
        for (const prop of props) {
            expect(hasPrevisPropBuilder(prop.id)).toBe(true);
            const parts = buildPrevisPropParts(prop.id);
            expect(parts.length).toBeGreaterThan(0);
            const size = previsPropPartsBounds(parts).getSize(new Vector3()).toArray();
            const reported = resolvePrevisPropSize(prop.id)!;
            size.forEach((value, axis) => {
                expect(Number.isFinite(value)).toBe(true);
                expect(value).toBeGreaterThan(0);
                expect(value).toBeCloseTo(reported[axis], 4);
            });
        }
    });

    test("未知 Shotloom 预设明确报错", () => {
        expect(() => buildPrevisPropParts("shotloom-does-not-exist")).toThrow();
    });

    test("生活道具的 5 毫米参数步进不会命中同一份几何缓存", () => {
        const first = previsPropPartsBounds(buildPrevisPropParts("shotloom-prop-cup", 0, { width: 0.085 })).getSize(new Vector3()).x;
        const next = previsPropPartsBounds(buildPrevisPropParts("shotloom-prop-cup", 0, { width: 0.09 })).getSize(new Vector3()).x;
        expect(first).toBeCloseTo(0.085, 5);
        expect(next).toBeCloseTo(0.09, 5);
    });

    test("宽度参数真正改变几何，开门参数包含子组变换", () => {
        const small = previsPropPartsBounds(buildPrevisPropParts("shotloom-shape-box", 0, { width: 2 }));
        const large = previsPropPartsBounds(buildPrevisPropParts("shotloom-shape-box", 0, { width: 7 }));
        expect(small.getSize(new Vector3()).x).toBeCloseTo(2);
        expect(large.getSize(new Vector3()).x).toBeCloseTo(7);
        const closed = previsPropPartsBounds(buildPrevisPropParts("shotloom-structure-door"));
        const opened = previsPropPartsBounds(buildPrevisPropParts("shotloom-structure-door", 0, { opening: 90 }));
        expect(opened.getSize(new Vector3()).z).toBeGreaterThan(closed.getSize(new Vector3()).z + 0.4);
    });

    test("家具承托点保留位置与角色，并隔离实例编辑", () => {
        const prop = PREVIS_PROP_CATALOG.find((item) => item.id === "shotloom-furniture-chair");
        expect(prop).toBeDefined();
        const anchors = resolveShotloomPropAnchors(prop!.id);
        expect(anchors.some((anchor) => anchor.role === "seat")).toBe(true);
        expect(anchors.every((anchor) => anchor.position.every(Number.isFinite))).toBe(true);
        anchors[0].position[1] = 999;
        expect(resolveShotloomPropAnchors(prop!.id)[0].position[1]).not.toBe(999);
    });

    test("帐篷布面从内部也可见", () => {
        expect(buildPrevisPropParts("shotloom-theme-tent").some((part) => previsPropMaterial(part.key).side === DoubleSide)).toBe(true);
    });

    test("源数值选项转成字符串枚举且后端镜像所有预设及参数", () => {
        const catalog = readFileSync(new URL("../../backend/internal/app/cloud_agent_previs_catalog.go", import.meta.url), "utf8");
        const params = readFileSync(new URL("../../backend/internal/app/cloud_agent_previs_params.go", import.meta.url), "utf8");
        for (const asset of SHOTLOOM_STATIC_ASSETS) {
            const id = `shotloom-${asset.id}`;
            expect(catalog).toContain(`{"${id}",`);
            const line = params.split("\n").find((item) => item.replace(/\s+/g, " ").includes(`"${id}": {`));
            expect(line).toBeDefined();
            const prop = PREVIS_PROP_CATALOG.find((item) => item.id === id)!;
            for (const [key, spec] of Object.entries(asset.parameters ?? {})) {
                const field = prop.params!.find((item) => item.key === key)!;
                if (spec.choices) {
                    expect(field.type).toBe("enum");
                    expect(field.default).toBe(String(spec.default));
                    expect(line).toContain(`previsEnum("${key}", "${spec.default}"`);
                } else {
                    expect(line).toContain(`previsNum("${key}", ${spec.min}, ${spec.max}, ${spec.default}, ${Boolean(spec.integer)})`);
                }
            }
        }
    });
});
