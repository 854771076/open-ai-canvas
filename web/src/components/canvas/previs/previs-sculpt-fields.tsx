import { Button, Checkbox, InputNumber, Select } from "antd";
import type { PrevisSculptPart, PrevisSculptSection } from "@/types/previs-sculpt";

function Points({ title, points, labels, min, max, onChange }: { title: string; points: number[][]; labels: string[]; min: number; max: number; onChange: (points: number[][]) => void }) {
    return (
        <details open className="previs-modeling__points">
            <summary>
                {title}（{points.length} 点）
            </summary>
            <div className="previs-modeling__note">各列：{labels.join(" / ")}，数值为部件尺寸的比例</div>
            {points.map((row, i) => (
                <div key={i} className="previs-modeling__point-row">
                    <span>{i + 1}</span>
                    {row.map((value, axis) => (
                        <InputNumber
                            key={axis}
                            size="small"
                            aria-label={`${title} ${i + 1} ${labels[axis]}`}
                            value={value}
                            step={0.05}
                            onChange={(next) => {
                                if (next !== null) onChange(points.map((p, j) => (j === i ? p.map((v, k) => (k === axis ? next : v)) : p)));
                            }}
                        />
                    ))}
                    <Button size="small" disabled={points.length <= min} aria-label={`删除${title}第 ${i + 1} 点`} onClick={() => onChange(points.filter((_, j) => i !== j))}>
                        删
                    </Button>
                </div>
            ))}
            <Button
                size="small"
                block
                disabled={points.length >= max}
                onClick={() => {
                    // 在最长线段中插入中点，避免新增重复端点或超出归一化范围。
                    let at = 0,
                        longest = -1;
                    for (let i = 0; i + 1 < points.length; i++) {
                        const length = points[i].reduce((sum, v, axis) => sum + (v - points[i + 1][axis]) ** 2, 0);
                        if (length > longest) {
                            longest = length;
                            at = i;
                        }
                    }
                    const point = points[at].map((v, axis) => (v + points[at + 1][axis]) / 2);
                    onChange([...points.slice(0, at + 1), point, ...points.slice(at + 1)]);
                }}
            >
                插入控制点
            </Button>
        </details>
    );
}

/** 曲线和截面使用结构化控件；高级 JSON 只补充细节、渐变和少见组合。 */
export function PrevisSculptFields({ part, onChange }: { part: PrevisSculptPart; onChange: (patch: Partial<PrevisSculptPart>) => void }) {
    const sectionChange = (index: number, patch: Partial<PrevisSculptSection>) => onChange({ sections: part.sections!.map((section, i) => (i === index ? ({ ...section, ...patch } as PrevisSculptSection) : section)) });
    return (
        <>
            {part.profile ? <Points title="旋转轮廓" labels={["半径", "高度"]} points={part.profile} min={3} max={32} onChange={(profile) => onChange({ profile: profile as [number, number][] })} /> : null}
            {part.path ? <Points title="管道路径" labels={["X", "Y", "Z"]} points={part.path} min={part.closed ? 3 : 2} max={32} onChange={(path) => onChange({ path: path as [number, number, number][] })} /> : null}
            {part.outline ? <Points title="挤出轮廓" labels={["X", "Y"]} points={part.outline} min={3} max={64} onChange={(outline) => onChange({ outline: outline as [number, number][] })} /> : null}
            {part.holes?.map((hole, i) => (
                <Points key={i} title={`孔洞 ${i + 1}`} labels={["X", "Y"]} points={hole} min={3} max={32} onChange={(points) => onChange({ holes: part.holes!.map((h, j) => (i === j ? (points as [number, number][]) : h)) })} />
            ))}
            {part.points ? <Points title="凸包顶点" labels={["X", "Y", "Z"]} points={part.points} min={4} max={48} onChange={(points) => onChange({ points: points as [number, number, number][] })} /> : null}
            {part.sections ? (
                <details open>
                    <summary>放样截面（{part.sections.length} 层）</summary>
                    {part.sections.map((section, i) => (
                        <div className="previs-modeling__section" key={i}>
                            <label>
                                第 {i + 1} 层高度
                                <InputNumber
                                    size="small"
                                    value={section.y}
                                    step={0.05}
                                    disabled={i === 0 || i === part.sections!.length - 1}
                                    onChange={(y) => {
                                        if (y !== null) sectionChange(i, { y });
                                    }}
                                />
                            </label>
                            {"points" in section ? (
                                <Points title={`截面 ${i + 1}`} labels={["X", "Z"]} points={section.points} min={3} max={24} onChange={(points) => sectionChange(i, { points: points as [number, number][] })} />
                            ) : (
                                <div className="previs-modeling__section-fields">
                                    {(["w", "d", "x", "z", "roundness"] as const).map((key) => (
                                        <label key={key}>
                                            {{ w: "宽度比例", d: "进深比例", x: "横向偏移", z: "纵向偏移", roundness: "圆方程度" }[key]}
                                            <InputNumber
                                                size="small"
                                                aria-label={`截面 ${i + 1} ${key}`}
                                                value={section[key] ?? (key === "roundness" ? 0.25 : 0)}
                                                step={0.05}
                                                onChange={(v) => {
                                                    if (v !== null) sectionChange(i, { [key]: v });
                                                }}
                                            />
                                        </label>
                                    ))}
                                </div>
                            )}
                            <Button size="small" disabled={i === 0 || i === part.sections!.length - 1} onClick={() => onChange({ sections: part.sections!.filter((_, j) => i !== j) })}>
                                删除该层
                            </Button>
                        </div>
                    ))}
                    <Button
                        size="small"
                        block
                        disabled={part.sections.length >= 12}
                        onClick={() => {
                            const sections = [...part.sections!];
                            let at = 0;
                            for (let i = 1; i + 1 < sections.length; i++) if (sections[i + 1].y - sections[i].y > sections[at + 1].y - sections[at].y) at = i;
                            sections.splice(at + 1, 0, { y: (sections[at].y + sections[at + 1].y) / 2, w: 0.5, d: 0.5 });
                            onChange({ sections });
                        }}
                    >
                        插入截面
                    </Button>
                </details>
            ) : null}
            {part.smooth !== undefined ? (
                <Checkbox checked={part.smooth} onChange={(e) => onChange({ smooth: e.target.checked })}>
                    平滑轮廓
                </Checkbox>
            ) : null}
            {part.closed !== undefined ? (
                <Checkbox checked={part.closed} onChange={(e) => onChange({ closed: e.target.checked })}>
                    闭合路径
                </Checkbox>
            ) : null}
            {part.facets !== undefined ? (
                <Checkbox checked={part.facets} onChange={(e) => onChange({ facets: e.target.checked })}>
                    保留平面棱角
                </Checkbox>
            ) : null}
            {part.shape === "scatter" ? (
                <>
                    <label>
                        散布物件
                        <Select
                            value={part.item}
                            options={[
                                ["blade", "草叶"],
                                ["tuft", "草簇"],
                                ["flower", "花朵"],
                                ["leaf", "树叶"],
                                ["pebble", "碎石"],
                                ["twig", "枝条"],
                            ].map(([value, label]) => ({ value, label }))}
                            onChange={(item) => onChange({ item })}
                        />
                    </label>
                    <label>
                        分布方式
                        <Select
                            value={part.spread ?? "ground"}
                            options={[
                                { value: "ground", label: "地面" },
                                { value: "shell", label: "椭球表面" },
                            ]}
                            onChange={(spread) => onChange({ spread })}
                        />
                    </label>
                </>
            ) : null}
        </>
    );
}
