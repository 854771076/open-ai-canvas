package app

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

var previsMeshColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Agent 与手动编辑器使用同一拓扑合同；只接受有界数据，不执行脚本或访问模型 URL。
func cloudAgentPrevisNormalizeMesh(input any) (map[string]any, error) {
	fail := func(field, message string) (map[string]any, error) {
		return nil, cloudAgentFieldError(field, "invalid_value", message+"；先读 manual=mesh")
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return fail("mesh", "网格必须是有效 JSON 对象")
	}
	var mesh struct {
		Version   float64  `json:"version"`
		Vertices  [][]any  `json:"vertices"`
		Triangles [][]any  `json:"triangles"`
		Colors    []string `json:"colors"`
	}
	if err := decodeCloudAgentJSONObject(string(raw), &mesh); err != nil {
		return fail("mesh", "网格只允许 version、vertices、triangles、colors，数值和数组类型必须正确")
	}
	if mesh.Version != 1 {
		return fail("mesh.version", "网格只支持版本 1")
	}
	if len(mesh.Vertices) < 3 || len(mesh.Vertices) > 20000 || len(mesh.Triangles) < 1 || len(mesh.Triangles) > 40000 {
		return fail("mesh", "网格需包含 3–20000 个顶点、1–40000 个三角面；整个场景仍受 1 MiB 上限约束")
	}
	vertices := make([][]float64, len(mesh.Vertices))
	for i, vertex := range mesh.Vertices {
		if len(vertex) != 3 {
			return fail(fmt.Sprintf("mesh.vertices[%d]", i), "顶点必须是三维米制坐标")
		}
		vertices[i] = make([]float64, 3)
		for axis, value := range vertex {
			// 先检查 JSON 数值类型；直接解码到 float64 会把 null 静默变成零，改写原拓扑。
			n, ok := value.(float64)
			if !ok {
				return fail(fmt.Sprintf("mesh.vertices[%d][%d]", i, axis), "顶点坐标必须是 JSON 数字，不能为 null 或其他类型")
			}
			if math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 10000 {
				return fail(fmt.Sprintf("mesh.vertices[%d]", i), "顶点坐标必须有限且绝对值不超过 10000 米")
			}
			vertices[i][axis] = n
		}
	}
	if len(mesh.Colors) != len(mesh.Triangles) {
		return fail("mesh.colors", "每个三角面必须对应一个 #rrggbb 颜色")
	}
	triangles := make([][]int, len(mesh.Triangles))
	for i, face := range mesh.Triangles {
		path := fmt.Sprintf("mesh.triangles[%d]", i)
		if len(face) != 3 {
			return fail(path, "每个三角面必须包含三个顶点索引")
		}
		triangle := make([]int, 3)
		for j, value := range face {
			n, ok := value.(float64)
			if !ok {
				return fail(fmt.Sprintf("%s[%d]", path, j), "三角面索引必须是 JSON 数字，不能为 null 或其他类型")
			}
			if math.IsNaN(n) || math.IsInf(n, 0) || math.Trunc(n) != n || n < 0 || n >= float64(len(mesh.Vertices)) {
				return fail(path, "三角面索引必须是有效顶点的零起始整数索引")
			}
			triangle[j] = int(n)
		}
		a, b, c := vertices[triangle[0]], vertices[triangle[1]], vertices[triangle[2]]
		u, v := [3]float64{}, [3]float64{}
		for axis := range u {
			u[axis], v[axis] = b[axis]-a[axis], c[axis]-a[axis]
		}
		x, y, z := u[1]*v[2]-u[2]*v[1], u[2]*v[0]-u[0]*v[2], u[0]*v[1]-u[1]*v[0]
		if x*x+y*y+z*z < 1e-16 {
			return fail(path, "三角面不能重复顶点或面积为零")
		}
		if !previsMeshColor.MatchString(mesh.Colors[i]) {
			return fail(fmt.Sprintf("mesh.colors[%d]", i), "面颜色必须是 #rrggbb")
		}
		mesh.Colors[i] = strings.ToLower(mesh.Colors[i])
		triangles[i] = triangle
	}
	return map[string]any{"version": 1, "vertices": vertices, "triangles": triangles, "colors": mesh.Colors}, nil
}

func cloudAgentPrevisMeshBounds(input any) (map[string]any, bool) {
	mesh, err := cloudAgentPrevisNormalizeMesh(input)
	if err != nil {
		return nil, false
	}
	min, max := []float64{math.Inf(1), math.Inf(1), math.Inf(1)}, []float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, vertex := range mesh["vertices"].([][]float64) {
		for axis, n := range vertex {
			min[axis], max[axis] = math.Min(min[axis], n), math.Max(max[axis], n)
		}
	}
	return map[string]any{"min": cloudAgentPrevisVec3(min), "max": cloudAgentPrevisVec3(max)}, true
}

func cloudAgentPrevisMeshSchema() map[string]any {
	triple := func(item map[string]any) map[string]any {
		return map[string]any{"type": "array", "minItems": 3, "maxItems": 3, "items": item}
	}
	return map[string]any{
		"type": "object", "description": "仅 kind=mesh；先读 manual=mesh。顶点/三角面/面颜色的完整拓扑，object_update 整体替换。",
		"properties": map[string]any{
			"version":   map[string]any{"type": "integer", "enum": []int{1}},
			"vertices":  map[string]any{"type": "array", "minItems": 3, "maxItems": 20000, "items": triple(map[string]any{"type": "number", "minimum": -10000, "maximum": 10000})},
			"triangles": map[string]any{"type": "array", "minItems": 1, "maxItems": 40000, "items": triple(map[string]any{"type": "integer", "minimum": 0, "maximum": 19999})},
			"colors":    map[string]any{"type": "array", "minItems": 1, "maxItems": 40000, "items": map[string]any{"type": "string", "pattern": "^#[0-9a-fA-F]{6}$"}},
		},
		"required": []string{"version", "vertices", "triangles", "colors"}, "additionalProperties": false,
	}
}
