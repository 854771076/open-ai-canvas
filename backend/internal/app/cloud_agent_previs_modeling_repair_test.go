package app

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"yingce/backend/internal/model"
)

func TestCloudAgentPrevisModelingErrorsAreRepairable(t *testing.T) {
	_, shapeErr := cloudAgentPrevisNormalizeSculpt(map[string]any{"parts": []any{map[string]any{"id": "spire", "shape": "cone", "size": []float64{1, 2, 1}}}})
	_, paramsErr := cloudAgentPrevisNormalizeParams("car", nil, map[string]any{"width": 2})
	for _, tc := range []struct {
		name, hint string
		err        error
	}{
		{"shape", "cylinder", shapeErr},
		{"params", "shotloom-vehicle-sedan", paramsErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var argumentErr *cloudAgentArgumentError
			var fieldErr *cloudAgentFieldArgumentError
			if !errors.As(tc.err, &argumentErr) || !errors.As(tc.err, &fieldErr) {
				t.Fatalf("建模参数错误不能交回模型修正: %v", tc.err)
			}
			if !strings.Contains(tc.err.Error(), tc.hint) {
				t.Fatalf("缺少可执行修正提示: %v", tc.err)
			}
		})
	}
}

func TestCloudAgentPrevisModelingRepairKeepsApprovalAndCanvasUnchanged(t *testing.T) {
	s, canvas := previsMutationFixture(t)
	policy, _ := s.RuntimePolicy()
	doc := mustCreationDocument(t, canvas.PayloadJSON)
	created, err := applyCloudAgentPrevisMutation(s.repo, "user", canvas.ID, previsMutationCall(t, "create", "previs_scene_create", map[string]any{
		"canvasSnapshotHash": cloudAgentCanvasHash(doc), "sceneId": "modeling", "title": "建模", "templateId": "empty",
	}), policy)
	if err != nil {
		t.Fatal(err)
	}
	hash := previsResultString(t, created, "snapshotHash")
	before, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	req := agentTestRequest()
	req.CanvasID, req.PermissionMode = canvas.ID, "request_approval"
	root, err := s.CreateCloudAgentRun("user", req, "")
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(operation map[string]any, callID string) (*model.CloudAgentExecution, cloudAgentRuntime) {
		t.Helper()
		run, state := agentInterjectionState(t, s, root.ID)
		call := previsMutationCall(t, callID, "previs_apply_patch", map[string]any{"snapshotHash": hash, "sceneId": "modeling", "operations": []any{operation}})
		state.Calls, state.CallIndex, state.ActiveTaskID = []cloudAgentCall{call}, 0, ""
		if err := s.advanceCloudAgentTool(run, &state); err != nil {
			t.Fatal(err)
		}
		return agentInterjectionState(t, s, root.ID)
	}
	failed, state := invoke(map[string]any{"type": "object_add", "id": "car-1", "kind": "prop", "propId": "car", "params": map[string]any{"width": 2}}, "bad-car")
	if failed.Status != "running" || state.Approval != nil || state.CallIndex != 1 {
		t.Fatalf("参数错误终止了会话或创建了审批: %s %#v", failed.Status, state.Approval)
	}
	message, ok := cloudAgentToolMessage(&state, "bad-car")
	var detail map[string]any
	if !ok || json.Unmarshal([]byte(message), &detail) != nil || detail["field"] != "params" || detail["reason"] != "invalid_tool_arguments" {
		t.Fatalf("缺少结构化纠错回执: %s", message)
	}
	waiting, state := invoke(map[string]any{"type": "object_add", "id": "car-1", "kind": "prop", "propId": "shotloom-vehicle-sedan", "params": map[string]any{"width": 2}}, "good-car")
	if waiting.Status != "waiting_approval" || state.Approval == nil {
		t.Fatalf("修正后未进入审批: %s %s", waiting.Status, waiting.FailureMessage)
	}
	after, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	if before.PayloadJSON != after.PayloadJSON {
		t.Fatal("参数错误或未审批的修正写入了画布")
	}
}
