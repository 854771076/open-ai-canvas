import type { Group } from "three";
import type { PrevisSculptRecipe } from "@/types/previs-sculpt";
export function buildSculptGroup(recipe: PrevisSculptRecipe, options?: { diagnostics?: { operation: string; cutterId: string; partId: string; message: string }[] }): Group;
export function disposeSculptGroup(group?: Group): void;
