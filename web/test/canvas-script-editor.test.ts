import { expect, test } from "bun:test";
import { resolve } from "node:path";

// 在隔离 DOM 中导入真实编辑器，避免 React 事件能力被其它测试的导入顺序改变。
test("分镜工作区通过阅读、单镜编辑、批量制作与脚本隔离回归", async () => {
    const child = Bun.spawn([process.execPath, "test", "./test/fixtures/canvas-script-editor.harness.tsx"], { cwd: resolve(import.meta.dir, ".."), timeout: 60000, stdout: "pipe", stderr: "pipe" });
    const [stdout, stderr, exitCode] = await Promise.all([new Response(child.stdout).text(), new Response(child.stderr).text(), child.exited]);
    const output = stdout + stderr;
    expect(exitCode, output).toBe(0);
    expect(output).toContain("17 pass");
    expect(output).toContain("0 fail");
}, 65000);
