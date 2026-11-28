package app

import (
	"fmt"
	"math"
	"strings"
)

// 世界引擎程序化道具与布景的白名单，ID 与 web/src/lib/canvas/previs/previs-prop-catalog.ts 一一对应。
// Agent 只能引用这里的 ID；几何体由前端按 ID 生成，后端不保存任何模型 URL。
type cloudAgentPrevisCatalogProp struct {
	ID       string
	Name     string
	Category string
	// Color 为强调部位的默认色（车身、花色、外墙）；空表示道具没有可变色部位。
	Color string
}

var cloudAgentPrevisPropCatalog = []cloudAgentPrevisCatalogProp{
	{"shotloom-shape-box", "方块", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-sphere", "球体", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-cylinder", "圆柱", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-cone", "圆锥", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-capsule", "胶囊", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-torus", "圆环", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-pyramid", "棱锥", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-plane", "平面板", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-wedge", "楔块", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-ramp", "斜板", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-arch", "拱形", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-hemisphere", "半球", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-tube", "空心管", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-l", "L 形块", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-u", "U 形块", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-shape-arc", "圆弧段", "基础形状(可调参数)", "#8795a5"},
	{"shotloom-furniture-chair", "餐椅", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-stool", "凳子", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-bench", "长凳", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-armchair", "扶手椅", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-sofa-single", "单人沙发", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-sofa", "多人沙发", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-barstool", "吧椅", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-lounger", "躺椅", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-table", "方桌", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-roundtable", "圆桌", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-desk", "书桌", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-coffeetable", "茶几", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-counter", "吧台", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-wardrobe", "衣柜 · 可开门", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-bookshelf", "书架", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-rack", "货架", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-bed-single", "单人床", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-bed-double", "双人床 · 可调尺寸", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-bunk", "上下铺", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-crib", "婴儿床", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-nightstand", "床头柜 · 抽屉轮廓", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-vanity", "梳妆台", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-screen", "屏风", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-coatstand", "衣帽架", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-cabinet", "橱柜", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-sink", "水槽", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-stove", "灶台", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-fridge", "冰箱", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-washer", "洗衣机", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-toilet", "马桶", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-bathtub", "浴缸", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-shower", "淋浴隔断", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-partition", "办公隔断", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-monitor", "电脑显示器", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-register", "收银台", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-lectern", "讲台", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-blackboard", "黑板", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-hospitalbed", "医院床", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-waitingbench", "候车座椅", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-furniture-vending", "自动售货机", "室内家具(可调参数)", "#8795a5"},
	{"shotloom-prop-cup", "杯子 · 空心", "生活道具(可调参数)", "#8795a5"},
	{"shotloom-prop-bowl", "碗", "生活道具(可调参数)", "#8795a5"},
	{"shotloom-prop-plate", "盘子", "生活道具(可调参数)", "#8795a5"},
	{"shotloom-prop-bottle", "瓶子", "生活道具(可调参数)", "#8795a5"},
	{"shotloom-prop-book", "书", "生活道具(可调参数)", "#8795a5"},
	{"shotloom-prop-phone", "手机", "生活道具(可调参数)", "#8795a5"},
	{"shotloom-prop-suitcase", "行李箱", "生活道具(可调参数)", "#8795a5"},
	{"shotloom-prop-backpack", "背包", "生活道具(可调参数)", "#8795a5"},
	{"shotloom-structure-wall", "墙 · 可留洞", "建筑构件(可调参数)", "#8795a5"},
	{"shotloom-structure-doorframe", "门框", "建筑构件(可调参数)", "#8795a5"},
	{"shotloom-structure-door", "门 · 带框开合", "建筑构件(可调参数)", "#8795a5"},
	{"shotloom-structure-windowframe", "窗框", "建筑构件(可调参数)", "#8795a5"},
	{"shotloom-structure-window", "窗 · 可开窗扇", "建筑构件(可调参数)", "#8795a5"},
	{"shotloom-structure-column", "立柱", "建筑构件(可调参数)", "#8795a5"},
	{"shotloom-structure-beam", "横梁", "建筑构件(可调参数)", "#8795a5"},
	{"shotloom-structure-slab", "楼板", "建筑构件(可调参数)", "#8795a5"},
	{"shotloom-structure-stairs-straight", "直楼梯", "楼梯与通道(可调参数)", "#8795a5"},
	{"shotloom-structure-stairs-l", "L 形楼梯", "楼梯与通道(可调参数)", "#8795a5"},
	{"shotloom-structure-stairs-u", "U 形楼梯", "楼梯与通道(可调参数)", "#8795a5"},
	{"shotloom-structure-stairs-crest", "双向升降楼梯", "楼梯与通道(可调参数)", "#8795a5"},
	{"shotloom-structure-stairs-spiral", "螺旋楼梯", "楼梯与通道(可调参数)", "#8795a5"},
	{"shotloom-structure-ramp", "坡道", "楼梯与通道(可调参数)", "#8795a5"},
	{"shotloom-structure-ladder", "梯子", "楼梯与通道(可调参数)", "#8795a5"},
	{"shotloom-structure-railing", "栏杆", "楼梯与通道(可调参数)", "#8795a5"},
	{"shotloom-road-straight", "直路", "道路市政(可调参数)", "#8795a5"},
	{"shotloom-road-curve", "弯路", "道路市政(可调参数)", "#8795a5"},
	{"shotloom-road-junction", "十字路口", "道路市政(可调参数)", "#8795a5"},
	{"shotloom-road-sidewalk", "人行道", "道路市政(可调参数)", "#8795a5"},
	{"shotloom-road-curb", "路缘石", "道路市政(可调参数)", "#8795a5"},
	{"shotloom-road-bridge", "桥面 · 带护栏", "道路市政(可调参数)", "#8795a5"},
	{"shotloom-road-busstop", "公交站亭", "道路市政(可调参数)", "#8795a5"},
	{"shotloom-road-parking", "停车位", "道路市政(可调参数)", "#8795a5"},
	{"shotloom-plant-broadleaf", "阔叶树", "花草树木(可调参数)", "#8795a5"},
	{"shotloom-plant-conifer", "针叶树", "花草树木(可调参数)", "#8795a5"},
	{"shotloom-plant-palm", "棕榈树", "花草树木(可调参数)", "#8795a5"},
	{"shotloom-plant-dead", "枯树", "花草树木(可调参数)", "#8795a5"},
	{"shotloom-plant-bush", "灌木", "花草树木(可调参数)", "#8795a5"},
	{"shotloom-plant-hedge", "树篱", "花草树木(可调参数)", "#8795a5"},
	{"shotloom-plant-grass", "草丛", "花草树木(可调参数)", "#8795a5"},
	{"shotloom-plant-potted", "盆栽", "花草树木(可调参数)", "#8795a5"},
	{"shotloom-terrain-ground", "地面块", "山石自然(可调参数)", "#8795a5"},
	{"shotloom-terrain-slope", "土坡", "山石自然(可调参数)", "#8795a5"},
	{"shotloom-terrain-hill", "丘陵块", "山石自然(可调参数)", "#8795a5"},
	{"shotloom-terrain-rock", "岩石 · 种子外形", "山石自然(可调参数)", "#8795a5"},
	{"shotloom-terrain-cliff", "岩壁", "山石自然(可调参数)", "#8795a5"},
	{"shotloom-terrain-ditch", "沟渠", "山石自然(可调参数)", "#8795a5"},
	{"shotloom-terrain-riverbed", "河床", "山石自然(可调参数)", "#8795a5"},
	{"shotloom-terrain-water", "水面占位", "山石自然(可调参数)", "#8795a5"},
	{"shotloom-building-bungalow", "平房", "建筑(可调参数)", "#8795a5"},
	{"shotloom-building-house", "两层住宅", "建筑(可调参数)", "#8795a5"},
	{"shotloom-building-apartment", "公寓体块", "建筑(可调参数)", "#8795a5"},
	{"shotloom-building-shop", "商铺", "建筑(可调参数)", "#8795a5"},
	{"shotloom-building-warehouse", "仓库", "建筑(可调参数)", "#8795a5"},
	{"shotloom-building-pavilion", "亭子", "建筑(可调参数)", "#8795a5"},
	{"shotloom-building-fortwall", "城墙", "建筑(可调参数)", "#8795a5"},
	{"shotloom-building-tower", "塔楼", "建筑(可调参数)", "#8795a5"},
	{"shotloom-industrial-container", "集装箱", "工业施工(可调参数)", "#8795a5"},
	{"shotloom-industrial-pallet", "托盘", "工业施工(可调参数)", "#8795a5"},
	{"shotloom-industrial-barrel", "油桶", "工业施工(可调参数)", "#8795a5"},
	{"shotloom-industrial-pipeline", "管线", "工业施工(可调参数)", "#8795a5"},
	{"shotloom-industrial-scaffold", "脚手架", "工业施工(可调参数)", "#8795a5"},
	{"shotloom-industrial-barrier", "施工围挡", "工业施工(可调参数)", "#8795a5"},
	{"shotloom-industrial-cone", "路锥", "工业施工(可调参数)", "#8795a5"},
	{"shotloom-industrial-handcart", "手推车", "工业施工(可调参数)", "#8795a5"},
	{"shotloom-theme-throne", "宝座", "古装舞台(可调参数)", "#8795a5"},
	{"shotloom-theme-table", "古式长案", "古装舞台(可调参数)", "#8795a5"},
	{"shotloom-theme-weaponrack", "兵器架", "古装舞台(可调参数)", "#8795a5"},
	{"shotloom-theme-flagpole", "旗杆", "古装舞台(可调参数)", "#8795a5"},
	{"shotloom-theme-altar", "祭坛", "古装舞台(可调参数)", "#8795a5"},
	{"shotloom-theme-ring", "擂台", "古装舞台(可调参数)", "#8795a5"},
	{"shotloom-theme-tent", "帐篷", "古装舞台(可调参数)", "#8795a5"},
	{"shotloom-theme-stall", "摊位", "古装舞台(可调参数)", "#8795a5"},
	{"shotloom-vehicle-sedan", "轿车", "交通工具(可调参数)", "#8795a5"},
	{"shotloom-vehicle-suv", "SUV", "交通工具(可调参数)", "#8795a5"},
	{"shotloom-vehicle-van", "面包车", "交通工具(可调参数)", "#8795a5"},
	{"shotloom-vehicle-bus", "公交车", "交通工具(可调参数)", "#8795a5"},
	{"shotloom-vehicle-truck", "卡车", "交通工具(可调参数)", "#8795a5"},
	{"shotloom-vehicle-bicycle", "自行车", "交通工具(可调参数)", "#8795a5"},
	{"shotloom-vehicle-motorcycle", "摩托车", "交通工具(可调参数)", "#8795a5"},
	{"shotloom-vehicle-boat", "小船", "交通工具(可调参数)", "#8795a5"},
	{"building", "楼房", "建筑(可调参数)", "#eeebe5"}, {"tower", "塔楼", "建筑(可调参数)", ""}, {"chinese_hall", "古建殿堂", "建筑(可调参数)", "#ece4d4"},
	{"road", "道路", "基础组件(可调参数)", ""}, {"wall", "墙体", "基础组件(可调参数)", ""}, {"ground_patch", "地面", "基础组件(可调参数)", ""},
	{"stairs", "台阶", "基础组件(可调参数)", ""}, {"fence_line", "围栏", "基础组件(可调参数)", ""}, {"room", "房间", "基础组件(可调参数)", "#ece8e1"},
	{"platform", "平台", "基础组件(可调参数)", ""}, {"column", "柱子", "基础组件(可调参数)", ""},
	{"tree_broadleaf", "阔叶树", "花草树木", ""}, {"tree_pine", "松树", "花草树木", ""}, {"tree_willow", "垂柳", "花草树木", ""},
	{"tree_cherry", "樱花树", "花草树木", "#f4a7c3"}, {"tree_palm", "棕榈树", "花草树木", ""}, {"bamboo", "竹丛", "花草树木", ""},
	{"bush", "灌木", "花草树木", ""}, {"hedge", "绿篱", "花草树木", ""}, {"grass", "草丛", "花草树木", ""},
	{"flowers", "野花丛", "花草树木", "#e85d8a"}, {"flower_bed", "花坛", "花草树木", "#e85d8a"}, {"potted_plant", "盆栽", "花草树木", "#b5653f"},
	{"lotus_pond", "荷花池", "花草树木", "#f4a7c3"},
	{"ancient_house", "古代民居", "古风建筑", "#ece4d4"}, {"ancient_shop", "古代商铺", "古风建筑", "#b8392c"}, {"ancient_courtyard", "四合院", "古风建筑", "#ece4d4"},
	{"tower_hall", "楼阁", "古风建筑", ""}, {"pagoda", "宝塔", "古风建筑", ""}, {"pavilion", "凉亭", "古风建筑", ""},
	{"city_gate", "城门", "古风建筑", ""}, {"ancient_wall", "城墙段", "古风建筑", ""}, {"paifang", "牌坊", "古风建筑", "#9e3a30"},
	{"stone_bridge", "石拱桥", "古风建筑", ""}, {"well", "水井", "古风建筑", ""}, {"market_stall", "集市摊位", "古风建筑", "#d9a441"},
	{"lantern_pole", "灯笼杆", "古风建筑", ""},
	{"modern_house", "现代住宅", "现代建筑", "#f1f1ee"}, {"villa", "坡顶别墅", "现代建筑", "#7a3e33"}, {"apartment", "公寓楼", "现代建筑", "#e3ddd3"},
	{"office_tower", "写字楼", "现代建筑", ""}, {"skyscraper", "摩天大楼", "现代建筑", ""}, {"shop_front", "临街商铺", "现代建筑", "#2f7de1"},
	{"street_lamp", "路灯", "街道设施", ""}, {"traffic_light", "红绿灯", "街道设施", ""}, {"bench", "长椅", "街道设施", "#a0714f"},
	{"bus_stop", "公交站", "街道设施", "#2f7de1"}, {"fence", "栅栏", "街道设施", ""}, {"trash_bin", "垃圾桶", "街道设施", "#3f7f5f"},
	{"car", "轿车", "交通工具", "#c0392b"}, {"bus", "公交车", "交通工具", "#2f7de1"}, {"ox_cart", "牛车", "交通工具", ""},
	{"bicycle", "自行车", "交通工具", "#d84949"},
	{"rock", "岩石", "山石自然", ""}, {"boulder", "巨石", "山石自然", ""}, {"hill", "山丘", "山石自然", ""}, {"log", "倒木", "山石自然", ""},
	{"sofa", "沙发", "室内家具", "#7d8ca3"}, {"dining_table", "餐桌", "室内家具", ""}, {"chair", "椅子", "室内家具", "#a0714f"},
	{"bed", "双人床", "室内家具", "#c9d6e3"}, {"bookshelf", "书架", "室内家具", ""}, {"desk", "书桌", "室内家具", ""},
}

type cloudAgentPrevisLayoutTemplate struct {
	ID       string
	Name     string
	Hint     string
	Default  float64
	Min, Max float64
}

var cloudAgentPrevisLayoutTemplates = []cloudAgentPrevisLayoutTemplate{
	{"ancient_city", "古代城池", "棋盘街巷、中轴楼阁、城门城墙", 420, 160, 1200},
	{"ancient_street", "古街集市", "一条长街，两侧商铺与摊位", 180, 80, 600},
	{"modern_city", "现代都市", "路网街区，中心高楼林立", 520, 200, 1200},
	{"modern_street", "城市街道", "双向车道、临街商铺与公寓", 200, 80, 600},
	{"village", "乡村", "散落民居、田地、土路与树林", 300, 100, 900},
	{"garden", "古典园林", "凉亭、拱桥、荷塘、花木", 110, 50, 300},
	{"forest", "森林", "松林阔叶林、山石与小径", 260, 80, 1000},
}

var (
	cloudAgentPrevisWeathers    = []string{"clear", "cloudy", "overcast", "rain", "storm", "snow", "fog"}
	cloudAgentPrevisGroundTypes = []string{"grass", "forest", "sand", "redsoil", "snow", "water", "city", "warm", "white", "gray"}
	cloudAgentPrevisLayoutIDs   = func() []string {
		ids := make([]string, 0, len(cloudAgentPrevisLayoutTemplates))
		for _, item := range cloudAgentPrevisLayoutTemplates {
			ids = append(ids, item.ID)
		}
		return ids
	}()
)

func cloudAgentPrevisFindProp(id string) (cloudAgentPrevisCatalogProp, bool) {
	for _, item := range cloudAgentPrevisPropCatalog {
		if item.ID == id {
			return item, true
		}
	}
	return cloudAgentPrevisCatalogProp{}, false
}

func cloudAgentPrevisFindLayoutTemplate(id string) (cloudAgentPrevisLayoutTemplate, bool) {
	for _, item := range cloudAgentPrevisLayoutTemplates {
		if item.ID == id {
			return item, true
		}
	}
	return cloudAgentPrevisLayoutTemplate{}, false
}

// cloudAgentPrevisCatalogGuide 放在 previs_scene_read 的返回里，而不是工具 schema 里：
// 只有读场景时才付这段 token，不占每一步都要发的固定 schema 预算。
func cloudAgentPrevisCatalogGuide() map[string]any {
	props := map[string][]string{}
	order := []string{}
	for _, item := range cloudAgentPrevisPropCatalog {
		if _, ok := props[item.Category]; !ok {
			order = append(order, item.Category)
		}
		props[item.Category] = append(props[item.Category], item.ID+" "+item.Name)
	}
	grouped := make(map[string]string, len(order))
	for _, category := range order {
		grouped[category] = strings.Join(props[category], "、")
	}
	layouts := make([]string, 0, len(cloudAgentPrevisLayoutTemplates))
	for _, item := range cloudAgentPrevisLayoutTemplates {
		layouts = append(layouts, fmt.Sprintf("%s %s（%s，%g-%gm）", item.ID, item.Name, item.Hint, item.Min, item.Max))
	}
	return map[string]any{
		"manual":        cloudAgentPrevisManualGuide(),
		"props":         grouped,
		"layouts":       strings.Join(layouts, "；"),
		"params":        cloudAgentPrevisParamsGuide(),
		"modeling":      "没有合适模型时自己造：先读 manual=sculpt 或 manual=mesh，再用 kind=sculpt 配方或 kind=mesh 三角拓扑。用户明确要求建模时不得用目录占位代替。params 只允许 catalog.params 中的道具；car 无参数，可调轿车用 shotloom-vehicle-sedan。",
		"cameraMoves":   cloudAgentPrevisCameraMovesGuide(),
		"cameraRecipes": cloudAgentPrevisCameraRecipesGuide(),
		"rules":         "目录已有实物优先 kind=prop+propId，原点在底面中心、y=0 贴地。没有合适模型时按 modeling 指引自己建模。params 仅按 catalog.params，不猜字段。数量明确的同类道具用 object_array_add（count+pattern+spacing+vary），例如三栋 12 层玻璃楼：propId=building,count=3,params={floors:12,style:glass}。成片街区、城镇、园林、森林用一个 kind=layout。外景先 world_update 设置时间、天气、远景。人物 A→B 移动用 actor_travel（travelStyle=walk/run/dash/jump/fly/sword_fly/float/teleport，time 起始秒，duration 用时，fromMarker/toMarker 为 scene.markers 的标签，如 A、B；没给起点则从演员当前位置出发），飞行类可给 height 米。不要用 actor_motion_add run 代替位移：那是原地动作。" + cloudAgentPrevisPathRules + cloudAgentPrevisCameraRules,
	}
}

func cloudAgentPrevisValidateLayout(layout map[string]any) error {
	template, ok := cloudAgentPrevisFindLayoutTemplate(stringValue(layout["template"]))
	if !ok {
		return BadAuthRequest("世界引擎布景模板不受支持")
	}
	if err := cloudAgentPrevisFiniteInRange(numberValue(layout["size"], math.NaN()), "布景范围", template.Min, template.Max); err != nil {
		return err
	}
	if err := cloudAgentPrevisFiniteInRange(numberValue(layout["density"], math.NaN()), "布景密度", 0.2, 1); err != nil {
		return err
	}
	if err := cloudAgentPrevisFiniteInRange(numberValue(layout["seed"], math.NaN()), "布景种子", 0, 999999999); err != nil {
		return err
	}
	if removed, ok := layout["removed"]; ok {
		items, isList := removed.([]any)
		if !isList || len(items) > 6000 {
			return BadAuthRequest("布景 removed 必须是不超过 6000 项的实例序号列表")
		}
		for _, value := range items {
			index := numberValue(value, math.NaN())
			if math.IsNaN(index) || index < 0 || index >= 6000 || index != math.Trunc(index) {
				return BadAuthRequest("布景 removed 含非法实例序号")
			}
		}
	}
	return nil
}

func cloudAgentPrevisValidateWorld(world map[string]any) error {
	if err := cloudAgentPrevisFiniteInRange(numberValue(world["timeOfDay"], math.NaN()), "世界时间", 0, 24); err != nil {
		return err
	}
	if err := cloudAgentPrevisValidateEnum(stringValue(world["weather"]), "世界天气", cloudAgentPrevisWeathers...); err != nil {
		return err
	}
	if err := cloudAgentPrevisValidateEnum(stringValue(world["ground"]), "世界远景", cloudAgentPrevisGroundTypes...); err != nil {
		return err
	}
	switch int(numberValue(world["groundSize"], 0)) {
	case 1000, 2000, 4000:
	default:
		return BadAuthRequest("世界地面范围只支持 1000/2000/4000 米")
	}
	return nil
}

// cloudAgentPrevisApplyWorld 处理 world_update：worldEnabled=false 关闭世界；否则在现有或默认值上合并。
func cloudAgentPrevisApplyWorld(scene map[string]any, operation cloudAgentPrevisPatchOperation) (string, error) {
	if operation.WorldEnabled != nil && !*operation.WorldEnabled {
		delete(scene, "world")
		return "关闭世界环境", nil
	}
	world, _ := scene["world"].(map[string]any)
	if world == nil {
		world = map[string]any{"timeOfDay": 10.0, "weather": "clear", "ground": "grass", "groundSize": 1000, "horizon": true}
	}
	if operation.TimeOfDay != nil {
		world["timeOfDay"] = *operation.TimeOfDay
	}
	if operation.Weather != "" {
		world["weather"] = operation.Weather
	}
	if operation.Ground != "" {
		world["ground"] = operation.Ground
	}
	if operation.GroundSize != nil {
		world["groundSize"] = *operation.GroundSize
	}
	if operation.Horizon != nil {
		world["horizon"] = *operation.Horizon
	}
	if err := cloudAgentPrevisValidateWorld(world); err != nil {
		return "", err
	}
	scene["world"] = world
	return fmt.Sprintf("世界环境：%s · %s · %s", stringValue(world["weather"]), stringValue(world["ground"]), formatPrevisHour(numberValue(world["timeOfDay"], 10))), nil
}

func formatPrevisHour(hours float64) string {
	total := int(math.Round(hours*60)) % (24 * 60)
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}

// cloudAgentPrevisAddCatalogObject 构造 kind=prop / kind=layout 对象；调用方已校验 ID 唯一。
func cloudAgentPrevisAddCatalogObject(operation cloudAgentPrevisPatchOperation, name string, position, rotation, scale []float64) (map[string]any, string, error) {
	item := map[string]any{"id": operation.ID, "kind": operation.Kind, "transform": map[string]any{"position": cloudAgentPrevisVec3(position), "rotation": cloudAgentPrevisVec3(rotation), "scale": cloudAgentPrevisVec3(scale)}, "visible": true, "castShadow": true, "receiveShadow": true, "keyframes": []any{}}
	switch operation.Kind {
	case "prop":
		prop, ok := cloudAgentPrevisFindProp(operation.PropID)
		if !ok {
			return nil, "", BadAuthRequest("propId 不在世界引擎道具目录中，先读 previs_scene_read 的 catalog")
		}
		if name == "" {
			name = prop.Name
		}
		color := prop.Color
		if color == "" {
			color = "#8795a5"
		}
		if operation.Color != nil {
			color = *operation.Color
		}
		params, err := cloudAgentPrevisNormalizeParams(prop.ID, nil, operation.Params)
		if err != nil {
			return nil, "", err
		}
		if params != nil {
			item["params"] = params
		}
		variant := 0
		if operation.Variant != nil {
			variant = *operation.Variant
		}
		if variant < 0 || variant > 7 {
			return nil, "", BadAuthRequest("variant 只支持 0-7")
		}
		item["propId"] = prop.ID
		item["variant"] = variant
		item["color"] = color
	case "layout":
		template, ok := cloudAgentPrevisFindLayoutTemplate(operation.LayoutTemplate)
		if !ok {
			return nil, "", BadAuthRequest("layoutTemplate 不受支持")
		}
		if name == "" {
			name = template.Name
		}
		layout := map[string]any{"template": template.ID, "seed": 1, "size": template.Default, "density": 0.75, "crowd": true}
		cloudAgentPrevisMergeLayout(layout, operation)
		if err := cloudAgentPrevisValidateLayout(layout); err != nil {
			return nil, "", err
		}
		item["layout"] = layout
		item["color"] = "#8795a5"
	}
	if err := cloudAgentPrevisValidateText(name, "对象名称", cloudAgentPrevisTextLimit, true); err != nil {
		return nil, "", err
	}
	item["name"] = name
	return item, name, nil
}

func cloudAgentPrevisMergeLayout(layout map[string]any, operation cloudAgentPrevisPatchOperation) {
	if operation.LayoutTemplate != "" {
		layout["template"] = operation.LayoutTemplate
	}
	if operation.Seed != nil {
		layout["seed"] = *operation.Seed
	}
	if operation.Size != nil {
		layout["size"] = *operation.Size
	}
	if operation.Density != nil {
		layout["density"] = *operation.Density
	}
	if operation.Crowd != nil {
		layout["crowd"] = *operation.Crowd
	}
}
