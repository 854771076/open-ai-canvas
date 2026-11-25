import { describe, expect, test } from "bun:test";
import { fileURLToPath } from "node:url";

import type { GenerationTaskEffectClaim } from "@/services/generation-task-materializer";

type WorkerRequest = { kind: "request"; id: number; action: string; args: unknown[] };
type WorkerResult = { kind: "result"; id: number; value?: unknown; error?: string };
type LockWaiter = { worker: Worker; id: number };

function deferred<T>() {
    let resolve!: (value: T) => void;
    const promise = new Promise<T>((done) => {
        resolve = done;
    });
    return { promise, resolve };
}

function sharedBrowserHarness(gateFirstRead = false) {
    const values = new Map<string, unknown>();
    const lockQueues = new Map<string, LockWaiter[]>();
    const workers: Worker[] = [];
    const firstRead = deferred<void>();
    let heldRead: (() => void) | undefined;
    let firstReadSeen = false;
    let lockRequests = 0;
    function reply(worker: Worker, id: number, value?: unknown) {
        worker.postMessage({ kind: "response", id, value });
    }
    const createRealm = () => {
        const worker = new Worker(new URL("./helpers/generation-effect-lock.worker.ts", import.meta.url).href, { type: "module" });
        workers.push(worker);
        let sequence = 0;
        const commands = new Map<number, { resolve: (value: unknown) => void; reject: (error: Error) => void }>();
        worker.onmessage = (event: MessageEvent<WorkerRequest | WorkerResult>) => {
            const message = event.data;
            if (message.kind === "result") {
                const command = commands.get(message.id);
                commands.delete(message.id);
                if (message.error) command?.reject(new Error(message.error));
                else command?.resolve(message.value);
                return;
            }
            const [name, value] = message.args as [string, unknown];
            if (message.action === "get") {
                const snapshot = values.get(name) ?? null;
                if (gateFirstRead && !firstReadSeen) {
                    firstReadSeen = true;
                    heldRead = () => reply(worker, message.id, snapshot);
                    firstRead.resolve();
                } else {
                    reply(worker, message.id, snapshot);
                    heldRead?.();
                    heldRead = undefined;
                }
            } else if (message.action === "set") {
                values.set(name, value);
                reply(worker, message.id, value);
            } else if (message.action === "lock") {
                lockRequests += 1;
                const queue = lockQueues.get(name) ?? [];
                queue.push({ worker, id: message.id });
                lockQueues.set(name, queue);
                if (queue.length === 1) reply(worker, message.id);
                else {
                    heldRead?.();
                    heldRead = undefined;
                }
            } else if (message.action === "unlock") {
                const queue = lockQueues.get(name)!;
                queue.shift();
                reply(worker, message.id);
                if (queue.length) reply(queue[0].worker, queue[0].id);
                else lockQueues.delete(name);
            }
        };
        worker.onerror = (event) => {
            for (const command of commands.values()) command.reject(new Error(event.message));
        };
        return {
            invoke<T = unknown>(action: string, ...args: unknown[]): Promise<T> {
                const id = ++sequence;
                return new Promise((resolve, reject) => {
                    commands.set(id, { resolve: resolve as (value: unknown) => void, reject });
                    worker.postMessage({ kind: "command", id, action, args });
                });
            },
        };
    };
    return {
        createRealm,
        firstRead: firstRead.promise,
        get lockRequests() {
            return lockRequests;
        },
        dispose: () => workers.forEach((worker) => worker.terminate()),
    };
}

describe("生成副作用跨标签租约", () => {
    test("两个独立页面同时领取只有一个owner，另一页面等待后可读到完成结果", async () => {
        const harness = sharedBrowserHarness(true);
        try {
            const first = harness.createRealm();
            const second = harness.createRealm();
            await Promise.all([first.invoke("initialize", { browserLocks: true, required: true }), second.invoke("initialize", { browserLocks: true, required: true })]);
            const claimingFirst = first.invoke<GenerationTaskEffectClaim>("claim", "materialize:shared-task:0", "shared-task");
            await harness.firstRead;
            const claimingSecond = second.invoke<GenerationTaskEffectClaim>("claim", "materialize:shared-task:0", "shared-task");
            const [owner, contender] = await Promise.all([claimingFirst, claimingSecond]);
            expect(owner.status).toBe("claimed");
            expect(contender.status).toBe("busy");
            if (owner.status !== "claimed") throw new Error("Expected a claimed lease");
            await expect(first.invoke("renew", "materialize:shared-task:0", "shared-task", owner.binding)).resolves.toEqual({ fence: owner.fence });
            await first.invoke("complete", "materialize:shared-task:0", "shared-task", { materializedAssetId: "saved-asset" }, owner.binding);
            await expect(second.invoke("claim", "materialize:shared-task:0", "shared-task")).resolves.toEqual({ status: "completed", result: { materializedAssetId: "saved-asset" } });
            expect(harness.lockRequests).toBe(5);
        } finally {
            harness.dispose();
        }
    });

    test("没有 Web Locks 时仍能在单标签互斥领取并持久化完成结果", async () => {
        const fixture = fileURLToPath(new URL("./fixtures/generation-effect-fallback.harness.ts", import.meta.url));
        const child = Bun.spawn([process.execPath, "test", fixture], { timeout: 20000, stdout: "pipe", stderr: "pipe" });
        const [stdout, stderr, exitCode] = await Promise.all([new Response(child.stdout).text(), new Response(child.stderr).text(), child.exited]);
        expect(exitCode, stdout + stderr).toBe(0);
    }, 25000);
});
