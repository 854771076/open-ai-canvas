import { expect, test } from "bun:test";
import localforage from "localforage";

import { createProviderNeutralGenerationTaskEffectStore } from "@/services/provider-neutral-generation-effects";

Object.defineProperty(globalThis, "window", { configurable: true, value: { localStorage: { getItem: () => "effect-owner" } } });
Object.defineProperty(globalThis, "document", { configurable: true, value: {} });
Object.defineProperty(globalThis, "navigator", { configurable: true, value: {} });

for (const required of [false, true]) {
    test(`无 Web Locks 时仍支持进程内互斥：requireCrossRealmLock=${required}`, async () => {
        const records = new Map<string, unknown>();
        localforage.getItem = (async (key: string) => records.get(key) ?? null) as typeof localforage.getItem;
        localforage.setItem = (async (key: string, value: unknown) => {
            records.set(key, value);
            return value;
        }) as typeof localforage.setItem;

        const options = { requireCrossRealmLock: required };
        const first = createProviderNeutralGenerationTaskEffectStore(options);
        const second = createProviderNeutralGenerationTaskEffectStore(options);
        const key = "materialize:task:0";
        const [owner, contender] = await Promise.all([first.claim(key, "task"), second.claim(key, "task")]);
        expect(owner.status).toBe("claimed");
        expect(contender.status).toBe("busy");
        if (owner.status !== "claimed") throw new Error("Expected a claimed lease");
        const result = { materializedAssetId: "saved-without-web-locks" };
        await first.complete(key, "task", result, owner.binding);
        await expect(second.claim(key, "task")).resolves.toEqual({ status: "completed", result });
        expect(records.size).toBe(1);
    });
}
