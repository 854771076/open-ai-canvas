import { Box3, Euler, Matrix4, Quaternion, Vector3 } from "three";
import type { PrevisObject, PrevisTransform } from "@/types/previs";
import { normalizeSculptRecipe, sculptBounds } from "./sculpt/sculpt-recipe.js";
import { previsMeshBounds, validatePrevisMesh } from "./previs-mesh";

const sculptCache = new WeakMap<object, NonNullable<PrevisObject["modelBounds"]>>();
const meshCache = new WeakMap<object, NonNullable<PrevisObject["modelBounds"]>>();

export function parsePrevisModelBounds(value: unknown): PrevisObject["modelBounds"] {
    if (!value || typeof value !== "object") return undefined;
    const bounds = value as { min?: unknown; max?: unknown };
    if (!Array.isArray(bounds.min) || !Array.isArray(bounds.max) || bounds.min.length !== 3 || bounds.max.length !== 3) return undefined;
    const min = bounds.min,
        max = bounds.max;
    if ([...min, ...max].some((n) => typeof n !== "number" || !Number.isFinite(n) || Math.abs(n) > 10000) || min.some((n, i) => n > max[i]) || min.every((n, i) => n === max[i])) return undefined;
    return { min: [...min] as [number, number, number], max: [...max] as [number, number, number] };
}

export function resolvePrevisModelBounds(object: Pick<PrevisObject, "modelBounds"> & Partial<Pick<PrevisObject, "kind" | "sculpt" | "mesh">>): PrevisObject["modelBounds"] {
    if (object.kind === "mesh" && object.mesh) {
        const cached = meshCache.get(object.mesh);
        if (cached) return cached;
        try {
            const bounds = previsMeshBounds(validatePrevisMesh(object.mesh));
            meshCache.set(object.mesh, bounds);
            return bounds;
        } catch {
            return undefined;
        }
    }
    if (object.kind !== "sculpt" || !object.sculpt) return object.modelBounds;
    const cached = sculptCache.get(object.sculpt);
    if (cached) return cached;
    const normalized = normalizeSculptRecipe(object.sculpt);
    if (!normalized.ok) return object.modelBounds;
    const bounds = sculptBounds(normalized.recipe) as NonNullable<PrevisObject["modelBounds"]>;
    sculptCache.set(object.sculpt, bounds);
    return bounds;
}

/** 保存的局部包围盒同时应用旋转与负缩放；取景、占地和贴地共用同一变换规则。 */
export function previsTransformedModelBox(object: Pick<PrevisObject, "modelBounds"> & Partial<Pick<PrevisObject, "kind" | "sculpt" | "mesh">>, transform: PrevisTransform, includePosition = true): Box3 | null {
    // Agent 更新配方时不信任旧派生包围盒，直接从新配方计算保守范围。
    const bounds = resolvePrevisModelBounds(object);
    if (!bounds || bounds.min.length !== 3 || bounds.max.length !== 3 || [...bounds.min, ...bounds.max].some((v) => !Number.isFinite(v)) || bounds.min.some((v, i) => v > bounds.max[i])) return null;
    const matrix = new Matrix4().compose(includePosition ? new Vector3(...transform.position) : new Vector3(), new Quaternion().setFromEuler(new Euler(...transform.rotation)), new Vector3(...transform.scale));
    return new Box3(new Vector3(...bounds.min), new Vector3(...bounds.max)).applyMatrix4(matrix);
}
