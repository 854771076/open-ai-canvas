import { Html } from "@react-three/drei";
import { useThree } from "@react-three/fiber";
import { useEffect, useRef, useState } from "react";
import type { Group } from "three";
import { applyPrevisSculptShadows, buildPrevisSculptGroup, disposePrevisSculptGroup } from "@/lib/canvas/previs/previs-sculpt";
import { buildPrevisMeshGroup } from "@/lib/canvas/previs/previs-mesh";
import type { PrevisObject } from "@/types/previs";

/** 配方和三角拓扑共用资源生命周期；无效输入只报告当前对象失败，保留上一份有效模型。 */
export function PrevisSculptVisual({ object }: { object: PrevisObject }) {
    const [display, setDisplay] = useState<Group>();
    const [error, setError] = useState("");
    const owned = useRef(new Set<Group>());
    const pending = useRef<Group | undefined>(undefined);
    const invalidate = useThree((state) => state.invalidate);
    useEffect(() => {
        if (object.kind === "mesh" ? !object.mesh : !object.sculpt) {
            setError("造型数据缺失");
            return;
        }
        try {
            const next = object.kind === "mesh" ? buildPrevisMeshGroup(object.mesh!) : buildPrevisSculptGroup(object.sculpt!);
            owned.current.add(next);
            pending.current = next;
            setDisplay(next);
            setError("");
            invalidate();
        } catch (reason) {
            setError(reason instanceof Error ? reason.message : "模型生成失败");
        }
    }, [object.kind, object.sculpt, object.mesh, invalidate]);
    // 新模型完成挂载后才释放旧资源，失败不会卸掉上一份有效对象。
    useEffect(() => {
        if (!display || display !== pending.current) return;
        for (const group of owned.current) {
            if (group === display) continue;
            disposePrevisSculptGroup(group);
            owned.current.delete(group);
        }
        applyPrevisSculptShadows(display, object.castShadow, object.receiveShadow);
        invalidate();
    }, [display, invalidate, object.castShadow, object.receiveShadow]);
    useEffect(
        () => () => {
            owned.current.forEach(disposePrevisSculptGroup);
            owned.current.clear();
        },
        [],
    );
    return (
        <>
            {display ? <primitive object={display} dispose={null} /> : null}
            {error ? (
                <Html center>
                    <span role="alert" className="previs-modeling__error">
                        {error}
                    </span>
                </Html>
            ) : null}
        </>
    );
}
