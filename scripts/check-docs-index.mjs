#!/usr/bin/env node
/**
 * 文档索引机器自检（human-docs 规范：退出码即判据）。
 *
 * 校验 docs/index.md：
 * 1. 防空集合——docs/ 下文档数异常少时直接失败，防止目录配错导致空集合全绿；
 * 2. 篇章齐且非空——每个 .md/.mdx 文件至少被索引引用一次，且无零字节文件；
 * 3. 链接可达——索引内每个相对链接（含 ../ 跨出 docs/ 的）都解析到真实文件；
 * 4. 技术模块编号 01..N 连续，且 backend 部分顺序与 backend/meta.json 一致；
 * 5. meta.json 与 backend/*.mdx 双向集合相等（导航不缺页、不引幽灵页）；
 * 6. 禁词扫描——索引描述文本不得出现日期与里程碑号（过程产物禁入）。
 *
 * 用法：bun scripts/check-docs-index.mjs
 */
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { dirname, join, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const docsRoot = join(repoRoot, 'docs');

const errors = [];
const warnings = [];
const fail = (msg) => errors.push(msg);

// ---- 收集 docs/ 下全部文档文件 ----
function walkDocs(dir) {
  const out = [];
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) {
      // 站点静态资源与构建产物不是篇章文档
      if (name === 'public' || name === 'node_modules' || name === '.next') continue;
      out.push(...walkDocs(p));
    } else if (/\.(md|mdx)$/.test(name)) {
      out.push(p);
    }
  }
  return out;
}

const relFromDocs = (p) => relative(docsRoot, p).split(sep).join('/');
const docFiles = walkDocs(docsRoot);
const docSet = new Map(docFiles.map((p) => [relFromDocs(p), p]));

if (docFiles.length < 10) fail(`docs/ 下仅发现 ${docFiles.length} 篇文档，疑似路径配置错误`);

// audit 防空集合守卫：主文档（NN-）少于 4 篇即视为目录配置错误，不能静默全绿
function auditFilesGuard(set) {
  const mains = [...set.keys()].filter((r) => /^\d{2}-/.test(r));
  if (mains.length < 4) {
    fail(`audit/ 主文档不足 4 篇（实见 ${mains.length}），疑似目录配置错误`);
    return false;
  }
  return true;
}

for (const [r, p] of docSet) {
  if (statSync(p).size === 0) fail(`空文档：${r}`);
}

// ---- 解析索引链接 ----
const indexPath = join(docsRoot, 'index.md');
const indexText = readFileSync(indexPath, 'utf8');

const LINK_RE = /\[([^\]]+)\]\(([^)\s]+)\)/g;
const links = [];
const refCount = new Map();
for (const m of indexText.matchAll(LINK_RE)) links.push({ text: m[1], target: m[2] });

if (links.length === 0) fail('索引中未解析到任何链接');

const referenced = new Set();
for (const { text, target } of links) {
  if (!text.trim()) fail(`链接文本为空：(${target})`);
  const clean = target.split('#')[0];
  if (!clean) continue; // 纯页内锚点，不判死链
  const normalized = relFromDocs(resolve(docsRoot, clean));
  refCount.set(normalized, (refCount.get(normalized) ?? 0) + 1);
  try {
    statSync(resolve(docsRoot, clean));
    if (docSet.has(normalized)) referenced.add(normalized);
  } catch {
    fail(`死链：${target}（链接文本「${text}」）`);
  }
}

// 每篇文档只收录一次为软约束：目录条目 + 正文交叉引用合法，但超过两次提示复核
for (const [r, n] of refCount) {
  if (docSet.has(r) && n > 2) warnings.push(`${r} 被引用 ${n} 次，请确认是否存在重复收录`);
}

// 全覆盖：index.md 自身不要求自引用
const missing = [...docSet.keys()].filter((r) => r !== 'index.md' && !referenced.has(r));
for (const r of missing) fail(`未被索引收录：${r}`);

// ---- 技术模块编号连续性 ----
const nums = [...indexText.matchAll(/^- \*\*(\d{2})\*\* /gm)].map((m) => Number(m[1]));
if (nums.length === 0) fail('未找到技术模块编号条目（**01** 样式）');
nums.forEach((n, i) => {
  if (n !== i + 1) fail(`技术模块编号不连续：第 ${i + 1} 个编号是 ${n}，应为 ${String(i + 1).padStart(2, '0')}`);
});

// ---- meta.json 与 backend 文档集一致性 ----
const numberedTargets = [...indexText.matchAll(/^- \*\*\d{2}\*\* \[[^\]]+\]\(([^)]+)\)/gm)].map((m) => m[1]);
const backendOrder = numberedTargets
  .map((t) => t.split('#')[0])
  .filter((t) => t.startsWith('content/docs/backend/'))
  .map((t) => t.replace('content/docs/backend/', '').replace(/\.mdx?$/, ''));

const metaPath = join(docsRoot, 'content/docs/backend/meta.json');
let metaPages = null;
try {
  metaPages = JSON.parse(readFileSync(metaPath, 'utf8')).pages;
} catch (e) {
  fail(`backend/meta.json 不可解析：${e.message}`);
}

if (metaPages) {
  const backendSlugs = [...docSet.keys()]
    .filter((r) => r.startsWith('content/docs/backend/') && r.endsWith('.mdx'))
    .map((r) => r.replace('content/docs/backend/', '').replace(/\.mdx$/, ''));
  const sameSet =
    metaPages.length === backendSlugs.length &&
    [...new Set(metaPages)].sort().join() === [...new Set(backendSlugs)].sort().join();
  if (!sameSet) {
    const onlyMeta = metaPages.filter((s) => !backendSlugs.includes(s));
    const onlyDisk = backendSlugs.filter((s) => !metaPages.includes(s));
    if (onlyMeta.length) fail(`meta.json 引用不存在的页面：${onlyMeta.join(', ')}`);
    if (onlyDisk.length) fail(`未进入 meta.json 导航的文档：${onlyDisk.join(', ')}`);
  }
  if (JSON.stringify(backendOrder) !== JSON.stringify(metaPages)) {
    fail('索引技术模块中 backend 文档的编号顺序与 meta.json pages 顺序不一致');
  }
}

// ---- 禁词扫描（过程产物禁入；只扫描述文本，文档标题身份保留）----
const prose = indexText.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, '$1').replace(/^\s*[-*]\s+/gm, '');
const dateHits = [...new Set([...prose.matchAll(/\d{4}-\d{2}-\d{2}/g)].map((m) => m[0]))];
if (dateHits.length) fail(`描述文本出现日期（过程产物禁入）：${dateHits.join(', ')}`);
const milestoneHits = [...new Set([...prose.matchAll(/\bM\d+\.\d+/g)].map((m) => m[0]))];
if (milestoneHits.length) fail(`描述文本出现里程碑号（过程产物禁入）：${milestoneHits.join(', ')}`);

// ---- audit/ 域自检（链接可达、子页被引用、零行号锚点、与 docs/ 隔离；fail-closed）----
const auditRoot = join(repoRoot, 'audit');
function walkAudit(dir) {
  const out = [];
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) out.push(...walkAudit(p));
    else if (/\.(md|mdx)$/.test(name)) out.push(p);
  }
  return out;
}
const auditRel = (p) => relative(auditRoot, p).split(sep).join('/');
const auditSet = new Map(walkAudit(auditRoot).map((p) => [auditRel(p), p]));

if (auditFilesGuard(auditSet)) {
  for (const [r, p] of auditSet) {
    if (statSync(p).size === 0) fail(`空文档：audit/${r}`);
  }
  const auditRefs = new Map();
  for (const [r, p] of auditSet) {
    const text = readFileSync(p, 'utf8');
    const lineHits = [...new Set([...text.matchAll(/\.(go|ts|tsx|mjs):\d+/g)].map((m) => m[0]))];
    if (lineHits.length) fail(`audit/${r} 出现行号锚点（人类阅读内容不用行号）：${lineHits.slice(0, 5).join(', ')}`);
    for (const m of text.matchAll(/\[([^\]]*)\]\(([^)\s]+)\)/g)) {
      const clean = m[2].split('#')[0];
      if (!clean || /^(https?:|mailto:)/.test(clean)) continue;
      if (/(^|[\s.(\/])(docs\/)/.test(clean) || clean.startsWith('../docs')) fail(`audit/${r} 引用 docs/（隔离断言）：${m[2]}`);
      const abs = resolve(auditRoot, clean);
      try {
        statSync(abs);
      } catch {
        fail(`audit 死链：${m[2]}（audit/${r}）`);
        continue;
      }
      const relNorm = relFromDocs(abs).startsWith('..') ? null : relFromDocs(abs);
      void relNorm;
      const relAudit = relative(auditRoot, abs).split(sep).join('/');
      if (auditSet.has(relAudit) && relAudit !== r) auditRefs.set(relAudit, (auditRefs.get(relAudit) ?? 0) + 1);
    }
  }
  for (const r of auditSet.keys()) {
    if (r.startsWith('details/') && !auditRefs.has(r)) fail(`details 子页未被任何主文档或 README 引用：audit/${r}`);
  }
}

// ---- 汇总 ----
if (errors.length) {
  console.error(`文档索引自检失败（${errors.length} 项）：`);
  for (const e of errors) console.error(`  ✗ ${e}`);
  process.exit(1);
}
console.log(
  `文档索引自检通过：${docFiles.length} 篇文档全覆盖且非空，${links.length} 个链接可达，` +
    `技术模块编号 01–${String(nums.length).padStart(2, '0')} 连续，meta.json 双向一致，禁词零命中。`
);
for (const w of warnings) console.log(`  ⚠ ${w}`);
