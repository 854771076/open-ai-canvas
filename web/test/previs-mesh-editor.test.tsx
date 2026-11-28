import { expect, test } from "bun:test";
import { resolve } from "node:path";

test("网格编辑器在 StrictMode 下保存、失败重试和卸载取消", () => {
    const result = Bun.spawnSync([process.execPath, "test", "./test/fixtures/previs-mesh-editor.harness.tsx"], { cwd: resolve(import.meta.dir, ".."), timeout: 15000 });
    const output = new TextDecoder().decode(result.stdout) + new TextDecoder().decode(result.stderr);
    expect(result.exitCode, output).toBe(0);
    expect(output).toContain("0 fail");
}, 20000);
