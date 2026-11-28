import { BoxGeometry, CylinderGeometry, Group, Mesh, MeshStandardMaterial, PlaneGeometry, SphereGeometry } from "three";
import { GLTFLoader } from "three-stdlib";
import { buildPrevisPropParts, previsPropMaterial } from "@/lib/canvas/previs/previs-prop-geometry";
import { previsObjectToEditableMesh } from "@/lib/canvas/previs/previs-mesh-conversion";
import { validatePrevisMesh } from "@/lib/canvas/previs/previs-mesh";
import { disposePrevisObject3D } from "@/lib/canvas/previs/previs-resources";
import { normalizeModel } from "@/lib/canvas/previs/previs-model-normalization";
import { buildPrevisSculptGroup, disposePrevisSculptGroup } from "@/lib/canvas/previs/previs-sculpt";
import { assertCurrentUserSession, captureUserSession } from "@/lib/user-scope";
import { getMediaBlob, resolveMediaUrl } from "./file-storage";
import type { PrevisObject } from "@/types/previs";

/** 原对象仍由场景持有；本函数只读取目录缓存，或释放自己加载的临时模型。 */
export async function loadPrevisMeshSource(object: PrevisObject, signal: AbortSignal) {
    signal.throwIfAborted();
    if (object.kind === "mesh") {
        if (!object.mesh) throw new Error("三角网格数据缺失");
        return structuredClone(validatePrevisMesh(object.mesh));
    }
    if (object.kind === "sculpt" && object.sculpt) {
        const root = buildPrevisSculptGroup(object.sculpt);
        try {
            return previsObjectToEditableMesh(root);
        } finally {
            disposePrevisSculptGroup(root);
        }
    }
    if (object.kind === "prop" && object.propId) {
        const root = new Group();
        buildPrevisPropParts(object.propId, object.variant, object.params).forEach((part) => root.add(new Mesh(part.geometry, previsPropMaterial(part.key, object.color))));
        return previsObjectToEditableMesh(root);
    }
    if (object.kind === "primitive" && object.primitive !== "character") {
        const geometry = object.primitive === "sphere" ? new SphereGeometry(0.6, 32, 24) : object.primitive === "cylinder" ? new CylinderGeometry(0.5, 0.5, 1.2, 32) : object.primitive === "plane" ? new PlaneGeometry(1.6, 1) : new BoxGeometry(1, 1, 1);
        const material = new MeshStandardMaterial({ color: object.color });
        try {
            return previsObjectToEditableMesh(new Mesh(geometry, material));
        } finally {
            geometry.dispose();
            material.dispose();
        }
    }
    if (object.kind !== "model") throw new Error("请选择静态道具、形状或模型，再复制为可编辑网格");
    const session = captureUserSession();
    let blob = object.storageKey ? await getMediaBlob(object.storageKey, signal) : null;
    if (!blob) {
        const url = await resolveMediaUrl(object.storageKey, object.url);
        if (!url) throw new Error("模型素材不存在");
        const response = await fetch(url, { signal, credentials: "same-origin" });
        if (!response.ok) throw new Error(`模型读取失败（${response.status}）`);
        blob = await response.blob();
    }
    signal.throwIfAborted();
    assertCurrentUserSession(session);
    const gltf = await new GLTFLoader().parseAsync(await blob.arrayBuffer(), "");
    try {
        signal.throwIfAborted();
        assertCurrentUserSession(session);
        if (gltf.animations.length) throw new Error("带动画的模型需先烘焙为静态模型");
        if (!object.meshEditable) normalizeModel(gltf.scene, object.castShadow, object.receiveShadow);
        return previsObjectToEditableMesh(gltf.scene);
    } finally {
        disposePrevisObject3D(gltf.scene);
    }
}
