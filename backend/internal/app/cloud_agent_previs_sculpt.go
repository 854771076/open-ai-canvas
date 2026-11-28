package app

// 本文件按 AGPL-3.0-or-later 移植 CozyClay 配方合同；来源与许可见 web/src/lib/canvas/previs/sculpt/SOURCE.md。
// 服务端只处理数据，不执行脚本或几何；浏览器严格生成通过后才能展示布尔结果。
import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

var previsSculptID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)
var previsSculptHex = regexp.MustCompile(`^[0-9a-fA-F]{3}([0-9a-fA-F]{3})?$`)
var previsSculptShapeKeys = map[string]string{
	"blob": "roundness taper", "box": "roundness", "cylinder": "taper", "torus": "", "frame": "border", "lathe": "profile smooth", "tube": "path thickness taper closed", "extrude": "outline holes roundness", "loft": "sections smooth", "convex": "points seed facetCount facets", "scatter": "item count seed spread",
}

type previsSculptFailure struct{ path, reason string }

func (e previsSculptFailure) Error() string { return e.path + ": " + e.reason }
func sculptFail(path, reason string)        { panic(previsSculptFailure{path, reason}) }
func sculptTidy(n float64) float64 {
	// 与前端一致：大数已经没有四位小数精度，避免舍入中间值溢出。
	if math.Abs(n) >= 1e12 {
		return n
	}
	return math.Floor(n*1e4+0.5) / 1e4
}
func sculptObject(v any, path string) map[string]any {
	m, ok := v.(map[string]any)
	if !ok || m == nil {
		sculptFail(path, "必须是对象")
	}
	return m
}
func sculptKeys(m map[string]any, allowed, path string) {
	for k := range m {
		if !sculptHasToken(allowed, k) {
			sculptFail(path+"."+k, "未知字段")
		}
	}
}
func sculptHasToken(allowed, value string) bool {
	// 必须逐项相等；包含空格的复合字段或枚举不能命中相邻的两个合法名称。
	for _, token := range strings.Fields(allowed) {
		if token == value {
			return true
		}
	}
	return false
}
func sculptNumber(v any, path string, min, max float64) float64 {
	n, ok := v.(float64)
	if !ok || math.IsNaN(n) || math.IsInf(n, 0) || n < min || n > max {
		sculptFail(path, fmt.Sprintf("必须是 %g–%g 的有限数字", min, max))
	}
	return sculptTidy(n)
}
func sculptUnit(m map[string]any, k, path string, fallback, min, max float64) float64 {
	v, ok := m[k]
	if !ok {
		return fallback
	}
	return sculptNumber(v, path+"."+k, min, max)
}
func sculptInt(m map[string]any, k, path string, fallback, min, max float64, required bool) float64 {
	v, ok := m[k]
	if !ok && !required {
		return fallback
	}
	n, ok := v.(float64)
	if !ok || math.Trunc(n) != n || n < min || n > max {
		sculptFail(path+"."+k, "整数超出允许范围")
	}
	return n
}
func sculptFlag(m map[string]any, k, path string, fallback bool) bool {
	v, ok := m[k]
	if !ok {
		return fallback
	}
	b, ok := v.(bool)
	if !ok {
		sculptFail(path+"."+k, "必须是布尔值")
	}
	return b
}
func sculptChoice(m map[string]any, k, path, fallback, choices string) string {
	v, exists := m[k]
	if !exists && fallback != "" {
		return fallback
	}
	s, ok := v.(string)
	if !ok || !sculptHasToken(choices, s) {
		sculptFail(path+"."+k, "枚举值无效")
	}
	return s
}
func sculptColor(v any, path string) string {
	s, ok := v.(string)
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if !ok || !previsSculptHex.MatchString(s) {
		sculptFail(path, "颜色必须是 #rgb 或 #rrggbb")
	}
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	return "#" + strings.ToLower(s)
}
func sculptList(v any, path string, min, max int) []any {
	a, ok := v.([]any)
	if !ok || len(a) < min || len(a) > max {
		sculptFail(path, fmt.Sprintf("数组长度必须为 %d–%d", min, max))
	}
	return a
}
func sculptPoints(v any, path string, min, max, dims int, lo, hi float64) [][]float64 {
	a := sculptList(v, path, min, max)
	out := make([][]float64, len(a))
	for i, p := range a {
		q := sculptList(p, fmt.Sprintf("%s[%d]", path, i), dims, dims)
		out[i] = make([]float64, dims)
		for j, n := range q {
			out[i][j] = sculptNumber(n, fmt.Sprintf("%s[%d][%d]", path, i, j), lo, hi)
		}
	}
	return out
}
func sculptVector(m map[string]any, k, path string, fallback []float64, lo, hi float64) []float64 {
	v, ok := m[k]
	if !ok && fallback != nil {
		return append([]float64(nil), fallback...)
	}
	return sculptPoints([]any{v}, path+"."+k, 1, 1, 3, lo, hi)[0]
}

func cloudAgentPrevisNormalizeSculpt(input any) (result map[string]any, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(previsSculptFailure); ok {
				result = nil
				err = cloudAgentFieldError("sculpt."+e.path, "invalid_value", "Sculpt "+e.Error())
			} else {
				panic(r)
			}
		}
	}()
	// 经 JSON 复制统一数字表示，避免直接调用与 HTTP 解码的类型差异，并防止修改调用方的配方。
	raw, marshalErr := json.Marshal(input)
	if marshalErr != nil {
		return nil, cloudAgentFieldError("sculpt", "invalid_json", "Sculpt 配方不是有效 JSON")
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil, cloudAgentFieldError("sculpt", "invalid_json", "Sculpt 配方不是有效 JSON")
	}
	root := sculptObject(value, "recipe")
	sculptKeys(root, "version parts", "recipe")
	if v, ok := root["version"]; ok && v != float64(1) {
		sculptFail("recipe.version", "只支持版本 1")
	}
	inputs := sculptList(root["parts"], "recipe.parts", 1, 48)
	parts := make([]map[string]any, len(inputs))
	byID := map[string]map[string]any{}
	for i, v := range inputs {
		p := sculptNormalizePart(v, i)
		id := p["id"].(string)
		if byID[id] != nil {
			sculptFail("parts["+id+"].id", "ID 重复")
		}
		parts[i] = p
		byID[id] = p
	}
	expanded, items := 0, 0.0
	for _, p := range parts {
		path := "parts[" + p["id"].(string) + "]"
		seen := map[string]bool{p["id"].(string): true}
		mirror := p["mirror"] == true
		ancestorMirror := false
		depth := 0
		for up := p; up["parent"] != nil; {
			id := up["parent"].(string)
			if seen[id] {
				sculptFail(path+".parent", "父链形成环")
			}
			seen[id] = true
			up = byID[id]
			if up == nil {
				sculptFail(path+".parent", "父部件不存在")
			}
			depth++
			if depth > 8 {
				sculptFail(path+".parent", "父链深度超过 8")
			}
			if up["mirror"] == true {
				ancestorMirror = true
			}
		}
		if mirror && ancestorMirror {
			sculptFail(path+".mirror", "镜像子树内不能嵌套镜像")
		}
		copies := 1
		if mirror || ancestorMirror {
			copies = 2
		}
		expanded += copies
		if p["shape"] == "scatter" {
			items += p["count"].(float64) * float64(copies)
		}
		parent := byID[stringValue(p["parent"])]
		if p["op"] != nil && parent == nil {
			sculptFail(path+".op", "切割体必须有父部件")
		}
		if parent != nil {
			if parent["op"] != nil {
				sculptFail(path+".parent", "切割体不能拥有子部件")
			}
			if p["op"] != nil && parent["shape"] == "scatter" {
				sculptFail(path+".op", "scatter 不能被切割")
			}
		}
	}
	if expanded > 48 {
		sculptFail("recipe.parts", "镜像展开超过 48 部件")
	}
	if items > 12000 {
		sculptFail("recipe.parts", "镜像展开超过 12000 个 scatter 实例")
	}
	return map[string]any{"version": 1, "parts": parts}, nil
}

func sculptNormalizePart(input any, index int) map[string]any {
	at := fmt.Sprintf("parts[%d]", index)
	m := sculptObject(input, at)
	all := "id shape size position rotation color parent mirror material finish detail gradient op"
	for _, keys := range previsSculptShapeKeys {
		all += " " + keys
	}
	sculptKeys(m, all, at)
	id, ok := m["id"].(string)
	if !ok || !previsSculptID.MatchString(id) {
		sculptFail(at+".id", "ID 必须为 1–40 位小写字母、数字或连字符")
	}
	path := "parts[" + id + "]"
	shape, ok := m["shape"].(string)
	keys, exists := previsSculptShapeKeys[shape]
	if !ok || !exists {
		sculptFail(path+".shape", "未知形状；可用 blob/box/cylinder/torus/frame/lathe/tube/extrude/loft/convex/scatter，圆锥用 cylinder+taper=1；先读 manual=sculpt")
	}
	for _, other := range previsSculptShapeKeys {
		for _, key := range strings.Fields(other) {
			if _, has := m[key]; has && !sculptHasToken(keys, key) {
				sculptFail(path+"."+key, "不适用于当前形状")
			}
		}
	}
	size := sculptVector(m, "size", path, nil, .005, 20)
	position := sculptVector(m, "position", path, []float64{0, 0, 0}, -20, 20)
	rotation := sculptVector(m, "rotation", path, []float64{0, 0, 0}, -math.MaxFloat64, math.MaxFloat64)
	for i, n := range rotation {
		n = math.Mod(math.Mod(n+180, 360)+360, 360) - 180
		if n == -180 {
			n = 180
		}
		rotation[i] = sculptTidy(n)
	}
	color := "#c2c6c8"
	if v, ok := m["color"]; ok {
		color = sculptColor(v, path+".color")
	}
	p := map[string]any{"id": id, "shape": shape, "size": size, "position": position, "rotation": rotation, "color": color}
	if v, ok := m["parent"]; ok {
		s, ok := v.(string)
		if !ok || s == "" {
			sculptFail(path+".parent", "父部件 ID 无效")
		}
		p["parent"] = s
	}
	if sculptFlag(m, "mirror", path, false) {
		p["mirror"] = true
	}
	for _, field := range []struct{ key, choices string }{{"material", "clay wood straw stone brick concrete metal glass ceramic plastic rubber fabric paper foliage"}, {"finish", "matte satin gloss"}} {
		if v, ok := m[field.key]; ok && v != nil {
			p[field.key] = sculptChoice(m, field.key, path, "", field.choices)
		}
	}
	if _, ok := m["op"]; ok {
		if p["parent"] == nil {
			sculptFail(path+".op", "切割体需要父部件")
		}
		p["op"] = sculptChoice(m, "op", path, "", "subtract intersect")
	}
	if v, ok := m["detail"]; ok {
		if shape == "scatter" {
			sculptFail(path+".detail", "scatter 不支持 detail")
		}
		d := sculptObject(v, path+".detail")
		sculptKeys(d, "amount scale seed", path+".detail")
		p["detail"] = map[string]any{"amount": sculptNumber(d["amount"], path+".detail.amount", 0, 1), "scale": sculptUnit(d, "scale", path+".detail", 4, 1, 12), "seed": sculptInt(d, "seed", path+".detail", 0, 0, 999999, false)}
	}
	if v, ok := m["gradient"]; ok {
		g := sculptObject(v, path+".gradient")
		sculptKeys(g, "color axis from to", path+".gradient")
		p["gradient"] = map[string]any{"color": sculptColor(g["color"], path+".gradient.color"), "axis": sculptChoice(g, "axis", path+".gradient", "y", "x y z"), "from": sculptUnit(g, "from", path+".gradient", 0, 0, 1), "to": sculptUnit(g, "to", path+".gradient", 1, 0, 1)}
	}
	switch shape {
	case "blob":
		p["roundness"] = sculptUnit(m, "roundness", path, .5, 0, 1)
		p["taper"] = sculptUnit(m, "taper", path, 0, -1, 1)
	case "box":
		p["roundness"] = sculptUnit(m, "roundness", path, .1, 0, 1)
	case "cylinder":
		p["taper"] = sculptUnit(m, "taper", path, 0, -1, 1)
	case "torus":
		if size[2] >= math.Min(size[0], size[1]) {
			sculptFail(path+".size", "圆环管径必须小于外径")
		}
	case "frame":
		limit := math.Min(size[0], size[1]) / 2
		b := sculptUnit(m, "border", path, sculptTidy(limit*.3), 0, limit)
		if b <= 0 || b >= limit {
			sculptFail(path+".border", "边框宽度必须大于 0 且小于短边一半")
		}
		p["border"] = b
	case "lathe":
		points := sculptPoints(m["profile"], path+".profile", 3, 32, 2, 0, 1)
		nonzero := false
		for i, q := range points {
			nonzero = nonzero || q[0] != 0
			if i > 0 && q[1] < points[i-1][1] {
				sculptFail(path+".profile", "轮廓高度必须递增")
			}
		}
		if !nonzero || points[0][1] == points[len(points)-1][1] {
			sculptFail(path+".profile", "轮廓退化")
		}
		p["profile"] = points
		p["smooth"] = sculptFlag(m, "smooth", path, true)
	case "tube":
		points := sculptPoints(m["path"], path+".path", 2, 32, 3, -.5, .5)
		thickness := sculptNumber(m["thickness"], path+".thickness", .005, math.Min(size[0], math.Min(size[1], size[2])))
		taper := sculptUnit(m, "taper", path, 0, -1, 1)
		closed := sculptFlag(m, "closed", path, false)
		if closed && (taper != 0 || len(points) < 3) {
			sculptFail(path+".path", "闭合管道至少三点且不能收尖")
		}
		for i, q := range points {
			if i == 0 && !closed {
				continue
			}
			before := points[(i+len(points)-1)%len(points)]
			distance := 0.
			for axis, n := range q {
				delta := (n - before[axis]) * (size[axis] - thickness)
				distance += delta * delta
			}
			if math.Sqrt(distance) < 1e-6 {
				sculptFail(path+".path", "拟合后相邻路径点重合")
			}
		}
		p["path"] = points
		p["thickness"] = thickness
		p["taper"] = taper
		p["closed"] = closed
	case "extrude":
		outline := sculptPoints(m["outline"], path+".outline", 3, 64, 2, -.5, .5)
		loops := [][][]float64{outline}
		if v, ok := m["holes"]; ok {
			for i, h := range sculptList(v, path+".holes", 0, 4) {
				loops = append(loops, sculptPoints(h, fmt.Sprintf("%s.holes[%d]", path, i), 3, 32, 2, -.5, .5))
			}
		}
		for _, loop := range loops {
			if math.Abs(sculptArea(loop)) < 1e-5 {
				sculptFail(path+".outline", "轮廓面积退化")
			}
		}
		if sculptLoopsCross(loops) {
			sculptFail(path+".outline", "轮廓自交或孔洞相交")
		}
		for i, hole := range loops[1:] {
			if !sculptInside(hole[0], outline) {
				sculptFail(path+".holes", "孔洞必须位于外轮廓内")
			}
			for j, other := range loops[1:] {
				if i != j && sculptInside(hole[0], other) {
					sculptFail(path+".holes", "孔洞不能嵌套")
				}
			}
		}
		p["outline"] = outline
		if len(loops) > 1 {
			p["holes"] = loops[1:]
		}
		p["roundness"] = sculptUnit(m, "roundness", path, .1, 0, 1)
	case "loft":
		sections := sculptList(m["sections"], path+".sections", 2, 12)
		out := make([]map[string]any, len(sections))
		last := -1.
		for i, v := range sections {
			at := fmt.Sprintf("%s.sections[%d]", path, i)
			s := sculptObject(v, at)
			y := sculptNumber(s["y"], at+".y", 0, 1)
			if y <= last {
				sculptFail(at+".y", "截面高度必须严格递增")
			}
			last = y
			next := map[string]any{"y": y}
			if points, ok := s["points"]; ok {
				sculptKeys(s, "y points", at)
				q := sculptPoints(points, at+".points", 3, 24, 2, -.5, .5)
				if math.Abs(sculptArea(q)) < 1e-6 || sculptLoftCross(q) {
					sculptFail(at+".points", "截面退化或自交")
				}
				next["points"] = q
			} else {
				sculptKeys(s, "y w d x z roundness", at)
				w := sculptNumber(s["w"], at+".w", .01, 1)
				d := sculptNumber(s["d"], at+".d", .01, 1)
				x := sculptUnit(s, "x", at, 0, -.5, .5)
				z := sculptUnit(s, "z", at, 0, -.5, .5)
				if math.Abs(x)+w/2 > .5+1e-9 || math.Abs(z)+d/2 > .5+1e-9 {
					sculptFail(at, "截面超出部件范围")
				}
				next["w"] = w
				next["d"] = d
				next["x"] = x
				next["z"] = z
				next["roundness"] = sculptUnit(s, "roundness", at, .25, 0, 1)
			}
			out[i] = next
		}
		if out[0]["y"] != float64(0) || last != 1 {
			sculptFail(path+".sections", "首尾截面高度必须是 0 和 1")
		}
		p["sections"] = out
		p["smooth"] = sculptFlag(m, "smooth", path, true)
	case "convex":
		hasPoints := m["points"] != nil
		hasSeed := m["seed"] != nil
		if hasPoints == hasSeed {
			sculptFail(path+".points", "必须且只能提供 points 或 seed")
		}
		if hasPoints {
			if m["facetCount"] != nil {
				v, has := m["seed"]
				if !has || v != nil {
					sculptFail(path+".facetCount", "只适用于种子岩石")
				}
			}
			points := sculptPoints(m["points"], path+".points", 4, 48, 3, -.5, .5)
			if sculptHullDepth(points) < .02 {
				sculptFail(path+".points", "凸包点必须有三维体积")
			}
			p["points"] = points
		} else {
			p["seed"] = sculptInt(m, "seed", path, 0, 0, 2147483647, true)
			if m["facetCount"] == nil {
				p["facetCount"] = float64(16)
			} else {
				p["facetCount"] = sculptInt(m, "facetCount", path, 16, 6, 40, true)
			}
		}
		p["facets"] = sculptFlag(m, "facets", path, true)
	case "scatter":
		p["item"] = sculptChoice(m, "item", path, "", "blade tuft flower leaf pebble twig")
		p["count"] = sculptInt(m, "count", path, 0, 1, 4000, true)
		p["seed"] = sculptInt(m, "seed", path, 0, 0, 999999, false)
		p["spread"] = sculptChoice(m, "spread", path, "ground", "ground shell")
	}
	return p
}

func sculptCross(a, b, c []float64) float64 { return (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0]) }
func sculptArea(q [][]float64) float64 {
	sum := 0.
	for i, a := range q {
		b := q[(i+1)%len(q)]
		sum += a[0]*b[1] - b[0]*a[1]
	}
	return sum / 2
}
func sculptInside(p []float64, q [][]float64) bool {
	inside := false
	for i, a := range q {
		b := q[(i+len(q)-1)%len(q)]
		if (a[1] > p[1]) != (b[1] > p[1]) && p[0] < (b[0]-a[0])*(p[1]-a[1])/(b[1]-a[1])+a[0] {
			inside = !inside
		}
	}
	return inside
}
func sculptSide(a, b, c []float64) float64 {
	n := sculptCross(a, b, c)
	if math.Abs(n) <= 1e-9 {
		return 0
	}
	return math.Copysign(1, n)
}
func sculptWithin(a, b, p []float64) bool {
	return math.Min(a[0], b[0])-1e-9 <= p[0] && p[0] <= math.Max(a[0], b[0])+1e-9 && math.Min(a[1], b[1])-1e-9 <= p[1] && p[1] <= math.Max(a[1], b[1])+1e-9
}
func sculptTouch(a, b, c, d []float64) bool {
	o1, o2, o3, o4 := sculptSide(a, b, c), sculptSide(a, b, d), sculptSide(c, d, a), sculptSide(c, d, b)
	return o1 != o2 && o3 != o4 || o1 == 0 && sculptWithin(a, b, c) || o2 == 0 && sculptWithin(a, b, d) || o3 == 0 && sculptWithin(c, d, a) || o4 == 0 && sculptWithin(c, d, b)
}
func sculptLoopsCross(loops [][][]float64) bool {
	for li, q := range loops {
		for i, a := range q {
			b := q[(i+1)%len(q)]
			for lj := li; lj < len(loops); lj++ {
				r := loops[lj]
				for j, c := range r {
					if li == lj && j <= i {
						continue
					}
					d := r[(j+1)%len(r)]
					if li == lj && ((i+1)%len(q) == j || (j+1)%len(q) == i) {
						shared, p, s := b, a, d
						if (j+1)%len(q) == i {
							shared, p, s = a, b, c
						}
						if sculptSide(shared, p, s) == 0 && (p[0]-shared[0])*(s[0]-shared[0])+(p[1]-shared[1])*(s[1]-shared[1]) > 0 {
							return true
						}
					} else if sculptTouch(a, b, c, d) {
						return true
					}
				}
			}
		}
	}
	return false
}
func sculptLoftCross(q [][]float64) bool {
	for i, a := range q {
		b := q[(i+1)%len(q)]
		for j := i + 2; j < len(q); j++ {
			if i == 0 && j == len(q)-1 {
				continue
			}
			c, d := q[j], q[(j+1)%len(q)]
			if sculptCross(a, b, c)*sculptCross(a, b, d) < 0 && sculptCross(c, d, a)*sculptCross(c, d, b) < 0 {
				return true
			}
		}
	}
	return false
}
func sculptHullDepth(q [][]float64) float64 {
	sub := func(a, b []float64) []float64 { return []float64{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
	cross := func(a, b []float64) []float64 {
		return []float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
	}
	length := func(a []float64) float64 { return math.Sqrt(a[0]*a[0] + a[1]*a[1] + a[2]*a[2]) }
	lo, hi := append([]float64(nil), q[0]...), append([]float64(nil), q[0]...)
	for _, p := range q {
		for k, n := range p {
			lo[k] = math.Min(lo[k], n)
			hi[k] = math.Max(hi[k], n)
		}
	}
	axis := 0
	for k := 1; k < 3; k++ {
		if hi[k]-lo[k] > hi[axis]-lo[axis] {
			axis = k
		}
	}
	a, b := q[0], q[0]
	for _, p := range q {
		if p[axis] < a[axis] {
			a = p
		}
		if p[axis] > b[axis] {
			b = p
		}
	}
	line := sub(b, a)
	denom := length(line)
	if denom == 0 {
		denom = 1
	}
	c := q[0]
	off := 0.
	for _, p := range q {
		n := length(cross(line, sub(p, a))) / denom
		if n > off {
			off = n
			c = p
		}
	}
	normal := cross(line, sub(c, a))
	denom = length(normal)
	if denom == 0 {
		denom = 1
	}
	plane := 0.
	for _, p := range q {
		v := sub(p, a)
		plane = math.Max(plane, math.Abs(v[0]*normal[0]+v[1]*normal[1]+v[2]*normal[2])/denom)
	}
	return math.Min(length(line), math.Min(off, plane))
}
