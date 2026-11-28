import { Box3, Mesh, Object3D, Vector3 } from "three";

/** 普通外部模型沿用两米最大边归一化；编辑器保存的米制 GLB 不走这个入口。 */
export function normalizeModel(root: Object3D, castShadow: boolean, receiveShadow: boolean) {
    root.updateMatrixWorld(true);
    const bounds = new Box3().setFromObject(root, true);
    const size = bounds.getSize(new Vector3());
    root.scale.multiplyScalar(2 / Math.max(size.x, size.y, size.z, 0.001));
    root.updateMatrixWorld(true);
    const centered = new Box3().setFromObject(root, true);
    const center = centered.getCenter(new Vector3());
    root.position.sub(center);
    root.position.y -= centered.min.y - center.y;
    root.traverse((child) => {
        if (!(child instanceof Mesh)) return;
        child.castShadow = castShadow;
        child.receiveShadow = receiveShadow;
    });
}
