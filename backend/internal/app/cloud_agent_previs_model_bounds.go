package app

import "math"

// GLB 网格的顶点保存在素材中，meshEditable 只标记这类模型；内联 kind=mesh 另行校验拓扑。
func cloudAgentPrevisValidateModelMetadata(object map[string]any) error {
	if value, exists := object["meshEditable"]; exists {
		editable, ok := value.(bool)
		if !ok || editable && (stringValue(object["kind"]) != "model" || stringValue(object["storageKey"]) == "" && stringValue(object["url"]) == "") {
			return BadAuthRequest("可编辑网格必须引用有效的静态模型素材")
		}
	}
	if value, exists := object["modelBounds"]; exists {
		bounds, ok := value.(map[string]any)
		if !ok || len(bounds) != 2 {
			return BadAuthRequest("模型包围盒格式无效")
		}
		min, minOK := cloudAgentPrevisNumberVec3(bounds["min"])
		max, maxOK := cloudAgentPrevisNumberVec3(bounds["max"])
		if !minOK || !maxOK {
			return BadAuthRequest("模型包围盒必须包含有限的三维坐标")
		}
		span := 0.0
		for i := range min {
			if math.Abs(min[i]) > 10000 || math.Abs(max[i]) > 10000 || min[i] > max[i] {
				return BadAuthRequest("模型包围盒超出范围或顺序错误")
			}
			span += max[i] - min[i]
		}
		if span <= 0 {
			return BadAuthRequest("模型包围盒不能为空")
		}
	}
	return nil
}

// Three.js XYZ 欧拉旋转对应先作用 Z、再 Y、最后 X；遍历八角点以覆盖旋转和负缩放。
func cloudAgentPrevisModelFloor(object, transform map[string]any) (float64, bool) {
	bounds, ok := object["modelBounds"].(map[string]any)
	if stringValue(object["kind"]) == "sculpt" {
		bounds, ok = cloudAgentPrevisSculptBounds(object["sculpt"])
	}
	if stringValue(object["kind"]) == "mesh" {
		bounds, ok = cloudAgentPrevisMeshBounds(object["mesh"])
	}
	if !ok {
		return 0, false
	}
	min, minOK := cloudAgentPrevisNumberVec3(bounds["min"])
	max, maxOK := cloudAgentPrevisNumberVec3(bounds["max"])
	rotation, rotationOK := cloudAgentPrevisNumberVec3(transform["rotation"])
	scale, scaleOK := cloudAgentPrevisNumberVec3(transform["scale"])
	if !minOK || !maxOK || !rotationOK || !scaleOK {
		return 0, false
	}
	cx, sx := math.Cos(rotation[0]), math.Sin(rotation[0])
	cy, sy := math.Cos(rotation[1]), math.Sin(rotation[1])
	cz, sz := math.Cos(rotation[2]), math.Sin(rotation[2])
	lowest := math.Inf(1)
	for mask := 0; mask < 8; mask++ {
		v := [3]float64{}
		for i := range v {
			value := min[i]
			if mask&(1<<i) != 0 {
				value = max[i]
			}
			v[i] = value * scale[i]
		}
		x, y, z := v[0]*cz-v[1]*sz, v[0]*sz+v[1]*cz, v[2]
		x, z = x*cy+z*sy, -x*sy+z*cy
		lowest = math.Min(lowest, y*cx-z*sx)
	}
	return -lowest, !math.IsInf(lowest, 0) && !math.IsNaN(lowest)
}

func cloudAgentPrevisGroundObjectTransform(object, transform map[string]any) {
	floor, hasBounds := cloudAgentPrevisModelFloor(object, transform)
	if !hasBounds {
		cloudAgentPrevisGroundPosition(transform)
		return
	}
	position, ok := cloudAgentPrevisNumberVec3(transform["position"])
	if ok && position[1] < floor-0.005 {
		position[1] = floor
		transform["position"] = cloudAgentPrevisVec3(position)
	}
}
