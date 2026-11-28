package app

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
)

func TestCloudAgentPrevisMeshSharedContract(t *testing.T) {
	raw, err := os.ReadFile("testdata/previs-mesh-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name  string
		Valid bool
		Field string
		Mesh  json.RawMessage
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var op cloudAgentPrevisPatchOperation
			if err := decodeCloudAgentJSONObject(`{"type":"object_add","id":"mesh-1","kind":"mesh","mesh":`+string(tc.Mesh)+`}`, &op); err != nil {
				t.Fatal(err)
			}
			objects := []map[string]any{}
			_, err := cloudAgentPrevisApplyObjectPatch(&objects, op, 0)
			if (err == nil) != tc.Valid {
				t.Fatalf("valid=%v err=%v", tc.Valid, err)
			}
			if err != nil {
				var argumentErr *cloudAgentArgumentError
				if !errors.As(err, &argumentErr) {
					t.Fatalf("网格错误不能修正: %v", err)
				}
				if tc.Field != "" {
					var fieldErr *cloudAgentFieldArgumentError
					if !errors.As(err, &fieldErr) || fieldErr.Issue != "invalid_value" || fieldErr.Field != tc.Field {
						t.Fatalf("null 元素缺少可纠正的网格字段错误: %v", err)
					}
				}
				if len(objects) != 0 {
					t.Fatal("非法网格已写入")
				}
				return
			}
			projected := cloudAgentPrevisSpatialLayout(map[string]any{"objects": objects})
			object := projected["objects"].([]map[string]any)[0]
			if object["mesh"] == nil || objects[0]["primitive"] != nil {
				t.Fatalf("网格缺少拓扑或仍是方块: %#v", object)
			}
		})
	}
}

func TestCloudAgentPrevisMeshPersistsReadsUpdatesAndRollsBack(t *testing.T) {
	s, canvas := previsMutationFixture(t)
	policy, _ := s.RuntimePolicy()
	doc := mustCreationDocument(t, canvas.PayloadJSON)
	created, err := applyCloudAgentPrevisMutation(s.repo, "user", canvas.ID, previsMutationCall(t, "create", "previs_scene_create", map[string]any{
		"canvasSnapshotHash": cloudAgentCanvasHash(doc), "sceneId": "mesh-scene", "title": "自建模型", "templateId": "empty",
	}), policy)
	if err != nil {
		t.Fatal(err)
	}
	hash := previsResultString(t, created, "snapshotHash")
	mesh := map[string]any{"version": 1, "vertices": [][]float64{{-1, 0, -1}, {1, 0, -1}, {0, 0, 1}, {0, 2, 0}}, "triangles": [][]int{{0, 1, 2}, {0, 3, 1}, {1, 3, 2}, {2, 3, 0}}, "colors": []string{"#cc8844", "#cc8844", "#cc8844", "#cc8844"}}
	patch := func(user, snapshot string, operations ...map[string]any) (any, error) {
		t.Helper()
		return applyCloudAgentPrevisMutation(s.repo, user, canvas.ID, previsMutationCall(t, "patch", "previs_apply_patch", map[string]any{"snapshotHash": snapshot, "sceneId": "mesh-scene", "operations": operations}), policy)
	}
	added, err := patch("user", hash, map[string]any{"type": "object_add", "id": "spire", "kind": "mesh", "name": "尖塔", "mesh": mesh})
	if err != nil {
		t.Fatal(err)
	}
	newHash := previsResultString(t, added, "snapshotHash")
	if hash == newHash {
		t.Fatal("创建网格未推进哈希")
	}
	readCall := previsMutationCall(t, "read", "previs_scene_read", map[string]any{"sceneId": "mesh-scene", "objectIds": []string{"spire"}, "includeTransforms": true})
	read, err := cloudAgentPrevisSceneRead(s.repo, "user", canvas.ID, readCall)
	if err != nil {
		t.Fatal(err)
	}
	scene := read.(map[string]any)["scene"].(map[string]any)
	spatial := scene["spatialLayout"].(map[string]any)["objects"].([]map[string]any)
	if len(spatial) != 1 || spatial[0]["mesh"] == nil {
		t.Fatalf("拓扑未按目标读回: %#v", spatial)
	}
	if scene["objects"].([]map[string]any)[0]["meshSummary"] == nil {
		t.Fatal("普通摘要缺少网格信息")
	}
	if _, err := cloudAgentPrevisSceneRead(s.repo, "other-user", canvas.ID, readCall); err == nil {
		t.Fatal("跨用户读取了网格")
	}
	if _, err := patch("other-user", newHash, map[string]any{"type": "object_update", "id": "spire", "mesh": mesh}); err == nil {
		t.Fatal("跨用户修改了网格")
	}
	if _, err := patch("user", hash, map[string]any{"type": "object_update", "id": "spire", "mesh": mesh}); err == nil {
		t.Fatal("旧哈希覆盖了网格")
	}
	mesh["vertices"].([][]float64)[3][1] = 4
	updated, err := patch("user", newHash, map[string]any{"type": "object_update", "id": "spire", "mesh": mesh})
	if err != nil {
		t.Fatal(err)
	}
	newHash = previsResultString(t, updated, "snapshotHash")
	before, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	storedScene, _ := findPrevisScene(creationMaps(mustCreationDocument(t, before.PayloadJSON)["previsScenes"]), "mesh-scene")
	object := cloudAgentPrevisFindByID(creationMaps(storedScene["objects"]), "spire")
	bounds := object["modelBounds"].(map[string]any)
	max, ok := cloudAgentPrevisNumberVec3(bounds["max"])
	if !ok || max[1] != 4 {
		t.Fatalf("修改网格仍使用旧包围盒: %#v", bounds)
	}
	_, err = patch("user", newHash,
		map[string]any{"type": "object_update", "id": "spire", "name": "不能保存"},
		map[string]any{"type": "object_update", "id": "spire", "mesh": nil},
	)
	if err == nil {
		t.Fatal("null 拓扑没有拒绝整批补丁")
	}
	after, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	if before.PayloadJSON != after.PayloadJSON {
		t.Fatal("失败补丁留下了部分修改")
	}
	if _, err := patch("user", newHash, map[string]any{"type": "object_remove", "id": "spire"}); err != nil {
		t.Fatal(err)
	}
}

func TestCloudAgentPrevisMeshRequiresApproval(t *testing.T) {
	s, canvas := previsMutationFixture(t)
	policy, _ := s.RuntimePolicy()
	doc := mustCreationDocument(t, canvas.PayloadJSON)
	created, err := applyCloudAgentPrevisMutation(s.repo, "user", canvas.ID, previsMutationCall(t, "create", "previs_scene_create", map[string]any{
		"canvasSnapshotHash": cloudAgentCanvasHash(doc), "sceneId": "mesh-scene", "title": "网格审批", "templateId": "empty",
	}), policy)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	req := agentTestRequest()
	req.CanvasID, req.PermissionMode = canvas.ID, "request_approval"
	root, err := s.CreateCloudAgentRun("user", req, "")
	if err != nil {
		t.Fatal(err)
	}
	run, state := agentInterjectionState(t, s, root.ID)
	mesh := map[string]any{"version": 1, "vertices": [][]float64{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}, "triangles": [][]int{{0, 1, 2}}, "colors": []string{"#ff0000"}}
	call := previsMutationCall(t, "mesh-add", "previs_apply_patch", map[string]any{"sceneId": "mesh-scene", "snapshotHash": previsResultString(t, created, "snapshotHash"), "operations": []any{map[string]any{"type": "object_add", "id": "mesh-1", "kind": "mesh", "mesh": mesh}}})
	state.Calls, state.CallIndex, state.ActiveTaskID = []cloudAgentCall{call}, 0, ""
	if err := s.advanceCloudAgentTool(run, &state); err != nil {
		t.Fatal(err)
	}
	waiting, state := agentInterjectionState(t, s, root.ID)
	if waiting.Status != "waiting_approval" || state.Approval == nil {
		t.Fatalf("网格未进入审批: %s %s", waiting.Status, waiting.FailureMessage)
	}
	after, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	if before.PayloadJSON != after.PayloadJSON {
		t.Fatal("审批前创建了网格")
	}
	if err := s.DecideCloudAgentApproval("user", root.ID, state.Approval.ID, "approve", "同意建模"); err != nil {
		t.Fatal(err)
	}
	if err := s.advanceCloudAgentByID("user", root.ID); err != nil {
		t.Fatal(err)
	}
	stored, _ := s.repo.CanvasProjectForUser("user", canvas.ID)
	if before.PayloadJSON == stored.PayloadJSON {
		t.Fatal("审批后未保存网格")
	}
}
