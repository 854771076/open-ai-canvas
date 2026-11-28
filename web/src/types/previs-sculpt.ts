import type { PrevisVec3 } from "./previs";

export type PrevisSculptShape = "blob" | "box" | "cylinder" | "torus" | "frame" | "lathe" | "tube" | "extrude" | "loft" | "convex" | "scatter";
export type PrevisSculptMaterial = "clay" | "wood" | "straw" | "stone" | "brick" | "concrete" | "metal" | "glass" | "ceramic" | "plastic" | "rubber" | "fabric" | "paper" | "foliage";
export type PrevisSculptSection = { y: number; points: [number, number][] } | { y: number; w: number; d: number; x?: number; z?: number; roundness?: number };
export type PrevisSculptPart = {
    id: string;
    shape: PrevisSculptShape;
    size: PrevisVec3;
    position?: PrevisVec3;
    rotation?: PrevisVec3;
    color?: string;
    parent?: string;
    mirror?: boolean;
    material?: PrevisSculptMaterial;
    finish?: "matte" | "satin" | "gloss";
    detail?: { amount: number; scale?: number; seed?: number };
    gradient?: { color: string; axis?: "x" | "y" | "z"; from?: number; to?: number };
    op?: "subtract" | "intersect";
    roundness?: number;
    taper?: number;
    border?: number;
    profile?: [number, number][];
    smooth?: boolean;
    path?: PrevisVec3[];
    thickness?: number;
    closed?: boolean;
    outline?: [number, number][];
    holes?: [number, number][][];
    sections?: PrevisSculptSection[];
    points?: PrevisVec3[];
    seed?: number;
    facetCount?: number;
    facets?: boolean;
    item?: "blade" | "tuft" | "flower" | "leaf" | "pebble" | "twig";
    count?: number;
    spread?: "ground" | "shell";
};
export type PrevisSculptRecipe = { version: 1; parts: PrevisSculptPart[] };
export type PrevisSculptResult = { ok: true; recipe: PrevisSculptRecipe } | { ok: false; path: string; error: string };
export type PrevisSculptEdit =
    | { add: PrevisSculptPart }
    | { remove: string }
    | ({ set: string } & Partial<Omit<PrevisSculptPart, "id" | "parent" | "material" | "finish" | "op" | "detail" | "gradient">> & { [K in "parent" | "material" | "finish" | "op" | "detail" | "gradient"]?: PrevisSculptPart[K] | null });
