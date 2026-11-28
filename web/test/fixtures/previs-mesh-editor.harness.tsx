import { afterAll, expect, mock, test } from "bun:test";
import { Window } from "happy-dom";
import { StrictMode, act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";

const dom = new Window({ url: "http://localhost/" });
Object.assign(globalThis, { window: dom, document: dom.document, navigator: dom.navigator, IS_REACT_ACT_ENVIRONMENT: true });
// 保留真实编辑器的状态和保存逻辑；隔离 WebGL 与外壳控件，测试只依赖 DOM 和生命周期。
mock.module("@react-three/fiber", () => ({
    Canvas: () => null,
    useThree: () => {
        throw new Error("测试不挂载 WebGL 场景");
    },
}));
mock.module("@react-three/drei", () => ({ OrbitControls: () => null, TransformControls: () => null }));
mock.module("@/components/ui/product/app-modal", () => ({ AppModal: ({ children }: { children: ReactNode }) => <div>{children}</div> }));
mock.module("antd", () => ({
    Button: ({ children, onClick, disabled, loading }: { children: ReactNode; onClick?: () => void; disabled?: boolean; loading?: boolean }) => (
        <button disabled={disabled || loading} onClick={onClick}>
            {children}
        </button>
    ),
    Alert: ({ title }: { title: string }) => <div role="alert">{title}</div>,
    Input: () => null,
    InputNumber: () => null,
    Segmented: () => null,
}));
const { PrevisMeshEditor } = await import("@/components/canvas/previs/previs-mesh-editor");
const { createPrevisMeshSeed } = await import("@/lib/canvas/previs/previs-mesh-conversion");
afterAll(() => dom.happyDOM.abort());

test("StrictMode 重放后仍可保存，失败解除锁定，重试使用新信号，卸载取消请求", async () => {
    const host = document.createElement("div");
    const root = createRoot(host);
    const signals: AbortSignal[] = [];
    let finish: (() => void) | undefined;
    await act(async () =>
        root.render(
            <StrictMode>
                <PrevisMeshEditor
                    initial={createPrevisMeshSeed()}
                    name="网格"
                    replacing={false}
                    onClose={() => {}}
                    onSave={async (_, __, signal) => {
                        signals.push(signal);
                        signal.throwIfAborted();
                        if (signals.length === 1) throw new Error("上传失败，请重试");
                        await new Promise<void>((resolve) => {
                            finish = resolve;
                        });
                    }}
                />
            </StrictMode>,
        ),
    );
    const save = () => [...host.querySelectorAll("button")].find((button) => button.textContent === "加入场景")!;
    try {
        await act(async () => save().click());
        expect(signals).toHaveLength(1);
        expect(signals[0].aborted).toBe(false);
        expect(host.querySelector('[aria-busy="false"]')).not.toBeNull();
        expect(host.querySelector('[role="alert"]')?.textContent).toContain("上传失败");
        expect(save().disabled).toBe(false);
        await act(async () => save().click());
        expect(signals).toHaveLength(2);
        expect(signals[1]).not.toBe(signals[0]);
        expect(signals[1].aborted).toBe(false);
        expect(host.querySelector('[aria-busy="true"]')).not.toBeNull();
        await act(async () => finish?.());
        expect(host.querySelector('[aria-busy="false"]')).not.toBeNull();
        expect(save().disabled).toBe(false);
        await act(async () => save().click());
        expect(signals).toHaveLength(3);
        await act(async () => root.unmount());
        expect(signals[2].aborted).toBe(true);
        await act(async () => finish?.());
    } finally {
        await act(async () => root.unmount());
    }
});
