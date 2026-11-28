import { SHAPE_ASSETS } from "./assets/catalog/shapes";
import { FURNITURE_ASSETS } from "./assets/catalog/furniture";
import { HAND_PROP_ASSETS } from "./assets/catalog/hand-props";
import { ARCHITECTURE_ASSETS, CIRCULATION_ASSETS } from "./assets/catalog/architecture";
import { ROAD_ASSETS } from "./assets/catalog/roads";
import { PLANT_ASSETS, TERRAIN_ASSETS } from "./assets/catalog/environment";
import { BUILDING_ASSETS } from "./assets/catalog/buildings";
import { INDUSTRIAL_ASSETS, THEMED_ASSETS, VEHICLE_ASSETS } from "./assets/catalog/production-props";

export const SHOTLOOM_STATIC_ASSETS = [
    ...SHAPE_ASSETS,
    ...FURNITURE_ASSETS,
    ...HAND_PROP_ASSETS,
    ...ARCHITECTURE_ASSETS,
    ...CIRCULATION_ASSETS,
    ...ROAD_ASSETS,
    ...PLANT_ASSETS,
    ...TERRAIN_ASSETS,
    ...BUILDING_ASSETS,
    ...INDUSTRIAL_ASSETS,
    ...THEMED_ASSETS,
    ...VEHICLE_ASSETS,
];
const byID = new Map(SHOTLOOM_STATIC_ASSETS.map((asset) => [asset.id, asset]));
export function findAsset(id: string) {
    return byID.get(id);
}
