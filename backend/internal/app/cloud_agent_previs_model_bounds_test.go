package app

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestCloudAgentPrevisSculptBoundsShared(t *testing.T) {
	raw, err := os.ReadFile("testdata/previs-sculpt-bounds.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name   string
		Input  any
		Bounds map[string]any
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			got, ok := cloudAgentPrevisSculptBounds(c.Input)
			if !ok {
				t.Fatal("valid recipe rejected")
			}
			a, _ := json.Marshal(got)
			b, _ := json.Marshal(c.Bounds)
			if string(a) != string(b) {
				t.Fatalf("bounds mismatch: %s %s", a, b)
			}
		})
	}
}

func TestCloudAgentPrevisModelMetadata(t *testing.T) {
	object := map[string]any{"kind": "model", "meshEditable": true, "storageKey": "model:test:mesh", "modelBounds": map[string]any{"min": []any{-1, -2, -3}, "max": []any{1, 2, 3}}}
	if err := cloudAgentPrevisValidateModelMetadata(object); err != nil {
		t.Fatal(err)
	}
	object["meshEditable"] = "yes"
	if err := cloudAgentPrevisValidateModelMetadata(object); err == nil {
		t.Fatal("invalid editable flag accepted")
	}
	object["meshEditable"] = true
	object["modelBounds"] = map[string]any{"min": []any{1, 0, 0}, "max": []any{0, 2, 3}}
	if err := cloudAgentPrevisValidateModelMetadata(object); err == nil {
		t.Fatal("reversed model bounds accepted")
	}
}

func TestCloudAgentPrevisModelGrounding(t *testing.T) {
	object := map[string]any{"modelBounds": map[string]any{"min": []any{-1, 0, -3}, "max": []any{1, 2, 3}}}
	transform := map[string]any{"position": []any{0, 0, 0}, "rotation": []any{0, 0, math.Pi / 2}, "scale": []any{-2, 1, 1}}
	cloudAgentPrevisGroundObjectTransform(object, transform)
	position, _ := cloudAgentPrevisNumberVec3(transform["position"])
	if math.Abs(position[1]-2) > 1e-8 {
		t.Fatalf("rotated negative-scale floor incorrect: %v", position)
	}
	transform["position"] = []any{0, 4, 0}
	cloudAgentPrevisGroundObjectTransform(object, transform)
	position, _ = cloudAgentPrevisNumberVec3(transform["position"])
	if position[1] != 4 {
		t.Fatal("elevated model unexpectedly dropped")
	}
}
