import type { Entity } from "../../model";
import { FURNITURE_SPECS } from "../catalog/furniture";
import { furnitureBuilder } from "./shared";
import { buildSeating } from "./seating";
import { buildStorage } from "./storage";
import { buildBedroom } from "./bedroom";
import { buildAppliance } from "./appliances";

export function makeFurniture(e: Entity) {
    const spec = FURNITURE_SPECS.find((s) => s.id === e.asset);
    if (!spec) throw new Error(`未知家具资产：${e.asset}`);
    const f = furnitureBuilder(e);
    if (![buildSeating, buildStorage, buildBedroom, buildAppliance].some((build) => build(f, spec.style))) throw new Error(`未实现家具结构：${spec.style}`);
    return f.root;
}
