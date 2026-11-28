package app

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCloudAgentPrevisSculptSharedContract(t *testing.T) {
	raw, err := os.ReadFile("testdata/previs-sculpt-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name       string `json:"name"`
		Valid      bool   `json:"valid"`
		Input      any    `json:"input"`
		Normalized any    `json:"normalized"`
	}
	if err = json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			result, err := cloudAgentPrevisNormalizeSculpt(f.Input)
			if (err == nil) != f.Valid {
				t.Fatalf("valid=%v result=%#v err=%v", f.Valid, result, err)
			}
			if err == nil {
				again, err := cloudAgentPrevisNormalizeSculpt(result)
				a, _ := json.Marshal(result)
				b, _ := json.Marshal(again)
				expected, _ := json.Marshal(f.Normalized)
				if string(a) != string(expected) {
					t.Fatalf("Go/JS normalized recipes differ:\ngot %s\nwant %s", a, expected)
				}
				if err != nil || string(a) != string(b) {
					t.Fatalf("normalization is not idempotent: %s %s %v", a, b, err)
				}
			}
		})
	}
}
func TestCloudAgentPrevisSculptBudgets(t *testing.T) {
	for _, mode := range []string{"parts", "depth", "scatter"} {
		t.Run(mode, func(t *testing.T) {
			parts := []any{}
			count := 49
			if mode == "depth" {
				count = 10
			}
			if mode == "scatter" {
				count = 4
			}
			for i := 0; i < count; i++ {
				id := strings.Repeat("a", i+1)
				if i > 39 {
					id = "z" + id[:39]
				}
				p := map[string]any{"id": id, "shape": "box", "size": []float64{1, 1, 1}}
				if mode == "depth" && i > 0 {
					p["parent"] = strings.Repeat("a", i)
				}
				if mode == "scatter" {
					p["shape"] = "scatter"
					p["count"] = 4000
					p["item"] = "blade"
				}
				parts = append(parts, p)
			}
			if _, err := cloudAgentPrevisNormalizeSculpt(map[string]any{"parts": parts}); err == nil {
				t.Fatal("budget overflow accepted")
			}
		})
	}
}
func TestCloudAgentPrevisSculptMutation(t *testing.T) {
	recipe := map[string]any{"parts": []any{map[string]any{"id": "body", "shape": "box", "size": []float64{1, 1, 1}}}}
	rawRecipe, _ := json.Marshal(recipe)
	objects := []map[string]any{}
	_, err := cloudAgentPrevisApplyObjectPatch(&objects, cloudAgentPrevisPatchOperation{Type: "object_add", ID: "sculpt-1", Kind: "sculpt", Sculpt: rawRecipe}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 1 || objects[0]["kind"] != "sculpt" || objects[0]["sculpt"] == nil {
		t.Fatalf("missing sculpt: %#v", objects)
	}
	projected := cloudAgentPrevisSpatialLayout(map[string]any{"objects": objects})
	items := projected["objects"].([]map[string]any)
	if items[0]["sculpt"] == nil {
		t.Fatal("includeTransforms omitted recipe")
	}
	_, err = cloudAgentPrevisApplyObjectPatch(&objects, cloudAgentPrevisPatchOperation{Type: "object_update", ID: "sculpt-1", Sculpt: json.RawMessage(`{"parts":[]}`)}, 0)
	if err == nil {
		t.Fatal("invalid update accepted")
	}
	_, err = cloudAgentPrevisApplyObjectPatch(&objects, cloudAgentPrevisPatchOperation{Type: "object_update", ID: "sculpt-1", Sculpt: json.RawMessage(`null`)}, 0)
	if err == nil {
		t.Fatal("explicit null recipe accepted")
	}
}
