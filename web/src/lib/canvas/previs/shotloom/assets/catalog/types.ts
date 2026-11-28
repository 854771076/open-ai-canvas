export interface AssetParameter {
    label: string;
    default: number;
    min: number;
    max: number;
    step: number;
    unit?: string;
    integer?: boolean;
    choices?: Readonly<Record<string, string>>;
}

/** 仅保留静态道具目录使用的字段，防止引入演员和动画依赖。 */
export interface AssetDefinition {
    id: string;
    name: string;
    kind: "prop";
    group: string;
    icon: string;
    aliases?: readonly string[];
    family?: string;
    capabilities?: { rig: "none"; actions: readonly never[]; pose: boolean; path: boolean };
    parameters?: Readonly<Record<string, AssetParameter>>;
}
