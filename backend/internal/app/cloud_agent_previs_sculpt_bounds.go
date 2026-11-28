package app

import "math"

// 配方坐标展开依据 CozyClay sculpt-recipe.js，按 AGPL-3.0-or-later 保留；见相邻 Sculpt 合同及其来源声明。
// 服务端仅计算部件盒的保守范围，不把这个结果当作布尔运算或精确碰撞检测。
func cloudAgentPrevisSculptBounds(input any) (map[string]any, bool) {
	recipe, err := cloudAgentPrevisNormalizeSculpt(input)
	if err != nil {
		return nil, false
	}
	children := map[string][]map[string]any{}
	for _, part := range creationMaps(recipe["parts"]) {
		parent := stringValue(part["parent"])
		children[parent] = append(children[parent], part)
	}
	min := []float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	max := []float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	compose := func(position, rotation []float64) [16]float64 {
		x, y, z := rotation[0]*math.Pi/180, rotation[1]*math.Pi/180, rotation[2]*math.Pi/180
		a, b, c, d, e, f := math.Cos(x), math.Sin(x), math.Cos(y), math.Sin(y), math.Cos(z), math.Sin(z)
		return [16]float64{c * e, a*f + b*e*d, b*f - a*e*d, 0, -c * f, a*e - b*f*d, b*e + a*f*d, 0, d, -b * c, a * c, 0, position[0], position[1], position[2], 1}
	}
	multiply := func(a, b [16]float64) [16]float64 {
		out := [16]float64{}
		for col := 0; col < 4; col++ {
			for row := 0; row < 4; row++ {
				for k := 0; k < 4; k++ {
					out[col*4+row] += a[k*4+row] * b[col*4+k]
				}
			}
		}
		return out
	}
	var visit func(map[string]any, [16]float64, bool)
	visit = func(part map[string]any, parent [16]float64, reflected bool) {
		position, _ := cloudAgentPrevisNumberVec3(part["position"])
		rotation, _ := cloudAgentPrevisNumberVec3(part["rotation"])
		if reflected {
			position[0] = -position[0]
			rotation[1], rotation[2] = -rotation[1], -rotation[2]
		}
		matrix := multiply(parent, compose(position, rotation))
		if part["op"] == nil {
			size, _ := cloudAgentPrevisNumberVec3(part["size"])
			for mask := 0; mask < 8; mask++ {
				p := [3]float64{}
				for axis := range p {
					p[axis] = size[axis] / 2
					if mask&(1<<axis) == 0 {
						p[axis] *= -1
					}
				}
				for axis := 0; axis < 3; axis++ {
					n := matrix[axis]*p[0] + matrix[4+axis]*p[1] + matrix[8+axis]*p[2] + matrix[12+axis]
					min[axis], max[axis] = math.Min(min[axis], n), math.Max(max[axis], n)
				}
			}
		}
		for _, child := range children[stringValue(part["id"])] {
			visit(child, matrix, reflected)
		}
		if !reflected && part["mirror"] == true {
			visit(part, parent, true)
		}
	}
	identity := compose([]float64{0, 0, 0}, []float64{0, 0, 0})
	for _, part := range children[""] {
		visit(part, identity, false)
	}
	for axis := range min {
		min[axis], max[axis] = sculptTidy(min[axis]), sculptTidy(max[axis])
	}
	return map[string]any{"min": cloudAgentPrevisVec3(min), "max": cloudAgentPrevisVec3(max)}, true
}
