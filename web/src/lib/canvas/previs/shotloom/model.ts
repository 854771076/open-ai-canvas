/** 静态道具构建器所需的最小输入，不依赖源编辑器的演员、时间轴或场景状态。 */
export interface Entity {
    asset: string;
    color: string;
    assetParameters?: Record<string, number>;
}
export type Vec3 = [number, number, number];
