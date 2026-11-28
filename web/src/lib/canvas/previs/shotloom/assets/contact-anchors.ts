import type { Vec3 } from "../model";

/** 保留源库承托点合同，世界坐标由调用方应用对象变换。 */
export interface ContactAnchor {
    id: string;
    role: "seat" | "surface" | "bed";
    position: Vec3;
    normal: Vec3;
    forward?: Vec3;
}
