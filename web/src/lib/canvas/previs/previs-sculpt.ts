import { Box3, type Group, Mesh } from "three";
import type { PrevisObject, PrevisVec3 } from "@/types/previs";
import type { PrevisSculptEdit, PrevisSculptRecipe, PrevisSculptResult } from "@/types/previs-sculpt";
import { normalizeSculptRecipe, patchSculptRecipe } from "./sculpt/sculpt-recipe.js";
import { buildSculptGroup, disposeSculptGroup } from "./sculpt/sculpt-geometry.js";
import { createPrevisObject } from "./previs-scene";

export type PrevisSculptBooleanDiagnostic = { operation: string; cutterId: string; partId: string; message: string };
export class PrevisSculptError extends Error {
    constructor(
        public readonly code: "RECIPE_INVALID" | "BOOLEAN_FAILED" | "GEOMETRY_FAILED",
        message: string,
        public readonly diagnostics: PrevisSculptBooleanDiagnostic[] = [],
    ) {
        super(`PREVIS_SCULPT_${code}: ${message}`);
    }
}
export function normalizePrevisSculptRecipe(input: unknown): PrevisSculptResult {
    return normalizeSculptRecipe(input) as PrevisSculptResult;
}
export function patchPrevisSculptRecipe(recipe: PrevisSculptRecipe, edits: PrevisSculptEdit[]): PrevisSculptResult & { changed?: string[] } {
    return patchSculptRecipe(recipe, edits) as PrevisSculptResult & { changed?: string[] };
}
export function buildPrevisSculptGroup(input: PrevisSculptRecipe): Group {
    const normalized = normalizePrevisSculptRecipe(input);
    if (!normalized.ok) throw new PrevisSculptError("RECIPE_INVALID", `${normalized.path}: ${normalized.error}`);
    const diagnostics: PrevisSculptBooleanDiagnostic[] = [];
    let group: Group | undefined;
    try {
        group = buildSculptGroup(normalized.recipe, { diagnostics });
        if (diagnostics.length) throw new PrevisSculptError("BOOLEAN_FAILED", diagnostics.map((d) => `${d.cutterId} → ${d.partId}: ${d.message}`).join("; "), diagnostics);
        group.traverse((node) => {
            if (!(node instanceof Mesh)) return;
            const positions = node.geometry.getAttribute("position");
            if (!positions?.count || !Array.from(positions.array).every(Number.isFinite)) throw new Error("模型几何为空或包含无效坐标");
        });
        const box = new Box3().setFromObject(group);
        if (box.isEmpty() || ![...box.min.toArray(), ...box.max.toArray()].every(Number.isFinite)) throw new Error("模型范围无效");
        return group;
    } catch (error) {
        if (group) disposeSculptGroup(group);
        if (error instanceof PrevisSculptError) throw error;
        throw new PrevisSculptError("GEOMETRY_FAILED", error instanceof Error ? error.message : String(error));
    }
}
const disposedSculptGroups = new WeakSet<Group>();
/** 对象开关不能覆盖玻璃的透光语义，重新开启投影时仍保留原生透明材质的例外。 */
export function applyPrevisSculptShadows(group: Group, castShadow: boolean, receiveShadow: boolean) {
    group.traverse((node) => {
        if (!(node instanceof Mesh)) return;
        const materials = Array.isArray(node.material) ? node.material : [node.material];
        const transmits = materials.some((material) => "transmission" in material && typeof material.transmission === "number" && material.transmission > 0);
        node.castShadow = castShadow && !transmits;
        node.receiveShadow = receiveShadow;
    });
}
export function disposePrevisSculptGroup(group: Group | undefined): void {
    if (!group || disposedSculptGroups.has(group)) return;
    disposedSculptGroups.add(group);
    disposeSculptGroup(group);
}
export function createPrevisSculptObject(input: PrevisSculptRecipe, name = "自由造型"): PrevisObject {
    const normalized = normalizePrevisSculptRecipe(input);
    if (!normalized.ok) throw new PrevisSculptError("RECIPE_INVALID", `${normalized.path}: ${normalized.error}`);
    const group = buildPrevisSculptGroup(normalized.recipe);
    try {
        const box = new Box3().setFromObject(group);
        return { ...createPrevisObject("box", name, [0, 0, 0]), kind: "sculpt", primitive: undefined, sculpt: normalized.recipe, modelBounds: { min: box.min.toArray() as PrevisVec3, max: box.max.toArray() as PrevisVec3 } };
    } finally {
        disposePrevisSculptGroup(group);
    }
}
