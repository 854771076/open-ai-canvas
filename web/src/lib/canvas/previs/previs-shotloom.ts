import { Box3, Color, DoubleSide, Matrix3, Mesh, MeshStandardMaterial, Vector3, type Group } from "three";
import { mergeGeometries } from "three/examples/jsm/utils/BufferGeometryUtils.js";
import type { PrevisPropParams, PrevisVec3 } from "@/types/previs";
import type { PrevisCatalogProp, PrevisPropCategoryId, PrevisPropParamDef } from "./previs-prop-catalog";
import type { PrevisPropMaterialKey, PrevisPropPart } from "./previs-prop-kit";
import { SHOTLOOM_STATIC_ASSETS, findAsset } from "./shotloom/asset-catalog";
import type { Entity } from "./shotloom/model";
import type { ContactAnchor } from "./shotloom/assets/contact-anchors";
import { makeShape } from "./shotloom/assets/shapes";
import { makeFurniture } from "./shotloom/assets/furniture";
import { makeHandProp } from "./shotloom/assets/hand-props";
import { makeArchitecture } from "./shotloom/assets/architecture";
import { makeCirculation } from "./shotloom/assets/circulation";
import { makeRoad } from "./shotloom/assets/roads";
import { makePlant } from "./shotloom/assets/plants";
import { makeTerrain } from "./shotloom/assets/terrain";
import { makeBuilding } from "./shotloom/assets/buildings";
import { makeIndustrial } from "./shotloom/assets/industrial";
import { makeThemed } from "./shotloom/assets/themed";
import { makeVehicle } from "./shotloom/assets/vehicles";

export const SHOTLOOM_PROP_PREFIX = "shotloom-";
const ACCENT = "#8795a5";
const BUILDERS: Record<string, (entity: Entity) => Group> = {
    "shape-v1": makeShape,
    "furniture-v1": makeFurniture,
    "hand-prop-v1": makeHandProp,
    "architecture-v1": makeArchitecture,
    "circulation-v1": makeCirculation,
    "road-v1": makeRoad,
    "plant-v1": makePlant,
    "terrain-v1": makeTerrain,
    "building-v1": makeBuilding,
    "industrial-v1": makeIndustrial,
    "themed-v1": makeThemed,
    "vehicle-v1": makeVehicle,
};
const CATEGORIES: Record<string, PrevisPropCategoryId> = {
    "shape-v1": "shape",
    "furniture-v1": "furniture",
    "hand-prop-v1": "hand_prop",
    "architecture-v1": "architecture",
    "circulation-v1": "circulation",
    "road-v1": "street",
    "plant-v1": "plant",
    "terrain-v1": "nature",
    "building-v1": "building",
    "industrial-v1": "industrial",
    "themed-v1": "themed",
    "vehicle-v1": "vehicle",
};

function sourceAsset(propID: string) {
    return propID.startsWith(SHOTLOOM_PROP_PREFIX) ? findAsset(propID.slice(SHOTLOOM_PROP_PREFIX.length)) : undefined;
}

export function hasShotloomPropBuilder(propID: string) {
    const asset = sourceAsset(propID);
    return Boolean(asset && BUILDERS[asset.family ?? ""]);
}

/** 数字选项在现有检查器内用字符串枚举，进入源构建器前恢复数字。 */
function sourceEntity(propID: string, params?: PrevisPropParams): Entity {
    const asset = sourceAsset(propID);
    if (!asset || !BUILDERS[asset.family ?? ""]) throw new Error(`未知 Shotloom 静态道具：${propID}`);
    const assetParameters: Record<string, number> = {};
    for (const [key, spec] of Object.entries(asset.parameters ?? {})) {
        const input = Number(params?.[key]);
        let value = Number.isFinite(input) ? Math.min(spec.max, Math.max(spec.min, input)) : spec.default;
        if (spec.integer) value = Math.round(value);
        if (spec.choices && !Object.hasOwn(spec.choices, String(value))) value = spec.default;
        assetParameters[key] = value;
    }
    return { asset: asset.id, color: ACCENT, assetParameters };
}

type Metadata = { size: PrevisVec3; anchors: ContactAnchor[] };
const metadataCache = new Map<string, Metadata>();
function metadataKey(entity: Entity) {
    return `${entity.asset}:${JSON.stringify(entity.assetParameters)}`;
}

/** 白膜主色映射强调色；阴影及固定色保留独立材质角色，兼容既有实例化渲染。 */
function materialKey(material: MeshStandardMaterial): PrevisPropMaterialKey {
    if (material.side === DoubleSide) return "roof";
    const color = material.color ?? new Color(ACCENT);
    if (color.getHexString() === new Color(ACCENT).getHexString()) return "accent";
    if (material.transparent && material.opacity < 1) return "glass";
    if (material.metalness > 0.3) return "metal";
    const hsl = color.getHSL({ h: 0, s: 0, l: 0 });
    if (hsl.l < 0.16) return "rubber";
    if (hsl.l > 0.82) return "white";
    if (hsl.s > 0.2 && hsl.h > 0.18 && hsl.h < 0.46) return "foliage";
    if (hsl.s > 0.35 && hsl.h < 0.15) return "wood";
    return "concrete_dark";
}

export function buildShotloomPropParts(propID: string, params?: PrevisPropParams): PrevisPropPart[] {
    const entity = sourceEntity(propID, params);
    const asset = findAsset(entity.asset)!;
    const root = BUILDERS[asset.family!](entity);
    root.updateMatrixWorld(true);
    const groups = new Map<PrevisPropMaterialKey, PrevisPropPart["geometry"][]>();
    const anchors: ContactAnchor[] = [];
    const materials = new Set<MeshStandardMaterial>();
    const originalGeometries = new Set<PrevisPropPart["geometry"]>();
    root.traverseVisible((object) => {
        const normalMatrix = new Matrix3().getNormalMatrix(object.matrixWorld);
        for (const anchor of (object.userData.contactAnchors ?? []) as ContactAnchor[]) {
            anchors.push({
                ...anchor,
                position: new Vector3(...anchor.position).applyMatrix4(object.matrixWorld).toArray(),
                normal: new Vector3(...anchor.normal).applyNormalMatrix(normalMatrix).toArray(),
                ...(anchor.forward ? { forward: new Vector3(...anchor.forward).transformDirection(object.matrixWorld).toArray() } : {}),
            });
        }
        if (!(object instanceof Mesh)) return;
        if (Array.isArray(object.material)) throw new Error(`Shotloom 道具使用了未适配的多材质网格：${propID}`);
        const material = object.material as MeshStandardMaterial;
        materials.add(material);
        originalGeometries.add(object.geometry);
        // 克隆并烘焙完整层级，门铰链、轮轴及拟合缩放才不会在扁平零件中丢失。
        const geometry = object.geometry.index ? object.geometry.toNonIndexed() : object.geometry.clone();
        geometry.applyMatrix4(object.matrixWorld);
        for (const name of Object.keys(geometry.attributes)) if (name !== "position" && name !== "normal") geometry.deleteAttribute(name);
        if (!geometry.getAttribute("normal")) geometry.computeVertexNormals();
        const key = materialKey(material);
        const bucket = groups.get(key) ?? [];
        bucket.push(geometry);
        groups.set(key, bucket);
    });
    const bounds = new Box3();
    const parts: PrevisPropPart[] = [];
    for (const [key, geometries] of groups) {
        const geometry = mergeGeometries(geometries, false);
        geometries.forEach((item) => item.dispose());
        if (!geometry) throw new Error(`Shotloom 几何合并失败：${propID}`);
        geometry.computeBoundingBox();
        geometry.computeBoundingSphere();
        geometry.userData.shotloomContactAnchors = anchors;
        bounds.union(geometry.boundingBox!);
        parts.push({ key, geometry });
    }
    originalGeometries.forEach((geometry) => geometry.dispose());
    materials.forEach((material) => material.dispose());
    // 源道路以路面为零点，厚度和桥墩伸到地下；目标目录统一底面原点，不能让新道具穿地。
    if (Math.abs(bounds.min.y) > 1e-5) {
        const lift = -bounds.min.y;
        parts.forEach((part) => {
            part.geometry.translate(0, lift, 0);
            part.geometry.computeBoundingBox();
            part.geometry.computeBoundingSphere();
        });
        anchors.forEach((anchor) => {
            anchor.position[1] += lift;
        });
        bounds.translate(new Vector3(0, lift, 0));
    }
    const size = bounds.getSize(new Vector3()).toArray();
    if (parts.length === 0 || !size.every((value) => Number.isFinite(value) && value > 0)) throw new Error(`Shotloom 几何尺寸无效：${propID}`);
    if (metadataCache.size >= 400) metadataCache.delete(metadataCache.keys().next().value!);
    metadataCache.set(metadataKey(entity), { size, anchors });
    return parts;
}

function metadata(propID: string, params?: PrevisPropParams): Metadata {
    const key = metadataKey(sourceEntity(propID, params));
    if (!metadataCache.has(key)) buildShotloomPropParts(propID, params).forEach((part) => part.geometry.dispose());
    return metadataCache.get(key)!;
}

export function resolveShotloomPropSize(propID: string, params?: PrevisPropParams): PrevisVec3 {
    return [...metadata(propID, params).size];
}

/** 返回副本，避免接触点编辑污染同预设的其他实例。 */
export function resolveShotloomPropAnchors(propID: string, params?: PrevisPropParams): ContactAnchor[] {
    return structuredClone(metadata(propID, params).anchors);
}

export const SHOTLOOM_PREVIS_PROPS: PrevisCatalogProp[] = SHOTLOOM_STATIC_ASSETS.map((asset) => ({
    id: SHOTLOOM_PROP_PREFIX + asset.id,
    name: asset.name,
    category: CATEGORIES[asset.family!],
    get size() {
        return resolveShotloomPropSize(SHOTLOOM_PROP_PREFIX + asset.id);
    },
    accent: "主体",
    params: Object.entries(asset.parameters ?? {}).map(([key, spec]): PrevisPropParamDef =>
        spec.choices
            ? {
                  key,
                  label: spec.label,
                  type: "enum",
                  default: String(spec.default),
                  options: Object.entries(spec.choices).map(([value, label]) => ({ value, label })),
              }
            : { key, label: spec.label, type: "number", min: spec.min, max: spec.max, step: spec.step, default: spec.default, unit: spec.unit },
    ),
}));
