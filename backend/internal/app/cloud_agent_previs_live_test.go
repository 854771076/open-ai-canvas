package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"yingce/backend/internal/model"
)

// 显式启用才访问真实模型；渠道凭据只从进程环境读取，不写入报告或测试输出。
func TestCloudAgentPrevisLiveModeling(t *testing.T) {
	if os.Getenv("CANVAS_AGENT_LIVE") != "1" {
		t.Skip("设置 CANVAS_AGENT_LIVE=1 并提供 OPENAI_BASE_URL/OPENAI_API_KEY 后运行真实模型验收")
	}
	base, key := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/"), os.Getenv("OPENAI_API_KEY")
	if base == "" || key == "" {
		t.Fatal("缺少真实模型接口配置")
	}
	modelKey := os.Getenv("CANVAS_AGENT_LIVE_MODEL")
	if modelKey == "" {
		modelKey = "gpt-6.2-sol"
	}
	root, err := filepath.Abs("../../../.local/project-workbench-debug/agent-mesh-live")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	reportDir, err := filepath.Abs("../../../.local/cache/agent-mesh/live")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(reportDir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AI_MODE", "live")
	t.Setenv("REDIS_URL", "")
	t.Setenv("YINGCE_AGENT_URL", "")
	t.Setenv("CANVAS_BACKEND_DATA_DIR", root)
	db, err := gorm.Open(sqlite.Open(filepath.Join(root, "live-"+newID()+".db")+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	fixture, _, _, _ := creationTestServiceOnDatabase(t, db)
	t.Cleanup(func() {
		// 即使初始化中途失败也清除渠道密钥；检查清理结果，避免把明文遗留误报为验收通过。
		for _, query := range []string{"PRAGMA secure_delete=ON", "UPDATE model_channels SET api_key = ''", "PRAGMA wal_checkpoint(TRUNCATE)", "VACUUM"} {
			if err := db.Exec(query).Error; err != nil {
				t.Errorf("清理本地模型凭据失败: %v", err)
			}
		}
	})
	profile := DefaultModelCapabilityConfigForModel(string(model.ChannelInterfaceOpenAIResponse), modelKey)
	if err := db.Model(&model.ModelChannel{}).Where("id = ?", "channel").Updates(map[string]any{"base_url": base, "api_key": key, "api_format": "openai"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.ChannelModel{}).Where("id = ?", "cm").Updates(map[string]any{"model_key": modelKey, "provider_model_key": modelKey, "protocol": model.ChannelInterfaceOpenAIResponse, "capability_config_json": mustEncodeModelCapabilityConfig(t, profile)}).Error; err != nil {
		t.Fatal(err)
	}
	s := New(fixture.repo, root)
	stop, workerDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(workerDone)
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = s.ProcessNextTask()
			select {
			case <-stop:
				return
			case <-time.After(30 * time.Millisecond):
			}
		}
	}()
	defer func() {
		close(stop)
		_ = s.Close()
		<-workerDone
	}()
	writeJSON := func(name string, value any) {
		t.Helper()
		raw, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(reportDir, name+".json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	canvasPayload := func(canvasID string) string {
		t.Helper()
		canvas, err := s.repo.CanvasProjectForUser("user", canvasID)
		if err != nil {
			t.Fatal(err)
		}
		return canvas.PayloadJSON
	}
	newCanvas := func(id string) {
		t.Helper()
		if err := db.Create(&model.CanvasProject{ID: id, UserID: "user", Title: "Agent 本地真实建模验收", PayloadJSON: `{"nodes":[],"connections":[],"previsScenes":[]}`}).Error; err != nil {
			t.Fatal(err)
		}
	}
	var runNumber int
	runAgent := func(t *testing.T, canvasID, prompt, permission, parent, decision string) *CloudAgentRun {
		t.Helper()
		runNumber++
		req := agentTestRequest()
		req.CanvasID, req.Prompt, req.Model, req.ChannelModelKey = canvasID, prompt, modelKey, modelKey
		req.PermissionMode, req.ReasoningMode = permission, "off"
		req.IdempotencyKey = fmt.Sprintf("live-mesh-%d-%d", time.Now().UnixNano(), runNumber)
		req.Budget.MaxSteps = 24
		run, err := s.CreateCloudAgentRun("user", req, parent)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("真实 Agent 开始：run=%s permission=%s model=%s", run.ID, permission, modelKey)
		before := canvasPayload(canvasID)
		deadline, since, approvals := time.Now().Add(5*time.Minute), 0, map[string]bool{}
		summaries := []map[string]any{}
		for time.Now().Before(deadline) {
			view, err := s.CloudAgentRun("user", run.ID, CloudAgentRunViewOptions{SinceSeq: since, FromStart: since == 0, EventLimit: 500})
			if err != nil {
				t.Fatal(err)
			}
			for _, event := range view.Events {
				if event.Seq <= since {
					continue
				}
				since = event.Seq
				summary := map[string]any{"seq": event.Seq, "type": event.Type, "runId": run.ID}
				for _, field := range []string{"toolName", "arguments", "text", "result", "reason", "errorClass", "approvalId"} {
					if v, ok := event.Payload[field]; ok {
						summary[field] = v
					}
				}
				summaries = append(summaries, summary)
				if event.Type == "tool_completed" || event.Type == "tool_failed" || event.Type == "run_failed" {
					t.Logf("run=%s event=%s tool=%s", run.ID, event.Type, stringValue(event.Payload["toolName"]))
				}
			}
			writeJSON(run.ID+"-events", summaries)
			if view.Status == "waiting_approval" && view.Approval != nil && !approvals[view.Approval.ID] {
				if canvasPayload(canvasID) != before {
					t.Fatal("审批前画布发生写入")
				}
				if decision == "" {
					t.Fatal("自动模式意外要求审批")
				}
				approvals[view.Approval.ID] = true
				t.Logf("run=%s 审批前未写入，执行本地验收决定=%s", run.ID, decision)
				if err := s.DecideCloudAgentApproval("user", run.ID, view.Approval.ID, decision, "本地真实建模能力验收"); err != nil {
					t.Fatal(err)
				}
				before = canvasPayload(canvasID)
			}
			if cloudAgentRunTerminal(view.Status) {
				writeJSON(run.ID+"-result", map[string]any{"runId": view.ID, "status": view.Status, "failure": view.FailureMessage, "model": modelKey, "steps": view.Step, "approvals": len(approvals)})
				writeJSON(canvasID+"-canvas", mustCreationDocument(t, canvasPayload(canvasID)))
				wantStatus := "completed"
				if decision == "reject" {
					// 拒绝审批会直接结束本轮，rejected 是成功阻止写入的预期终态。
					wantStatus = "rejected"
				}
				if view.Status != wantStatus {
					t.Fatalf("真实 Agent 失败：status=%s reason=%s", view.Status, view.FailureMessage)
				}
				if permission == "request_approval" && len(approvals) == 0 {
					t.Fatal("未覆盖真实审批")
				}
				view.Events = nil
				for _, summary := range summaries {
					view.Events = append(view.Events, CloudAgentEvent{Type: stringValue(summary["type"]), Payload: summary})
				}
				return view
			}
			if view.Status == "waiting_user" {
				t.Fatal("精确测试请求意外停在用户提问，详见事件报告")
			}
			time.Sleep(300 * time.Millisecond)
		}
		_ = s.CancelCloudAgent(context.Background(), "user", run.ID)
		t.Fatal("真实 Agent 超时，已取消；详见事件报告")
		return nil
	}

	t.Run("create-and-update-mesh", func(t *testing.T) {
		id := "live-mesh-canvas"
		newCanvas(id)
		created := runAgent(t, id, "这是本地建模验收。当前没有现成模型，请自己造：创建一个空白世界引擎场景，再用新增三角网格建模做一座科幻信标塔，物体名称为验收信标。要求一个真实三角网格对象，由四条支腿、两层方形平台、八棱锥尖顶共同组成，不要用目录道具或单个基础体替代。局部地面为0米，总高4米；至少20个顶点、24个三角面；支架灰色、尖顶红色。场景位置[2,0,-3]。自己读取工具说明再执行，完成后读回拓扑核对。无需视频预览、截图、付费生成或向我提问，只做建模。", "auto", "", "")
		doc := mustCreationDocument(t, canvasPayload(id))
		var scene, object map[string]any
		for _, candidate := range creationMaps(doc["previsScenes"]) {
			for _, item := range creationMaps(candidate["objects"]) {
				if stringValue(item["kind"]) == "mesh" {
					scene, object = candidate, item
				}
			}
		}
		if object == nil {
			t.Fatal("真实 Agent 未使用三角网格")
		}
		mesh, err := cloudAgentPrevisNormalizeMesh(object["mesh"])
		if err != nil {
			t.Fatal(err)
		}
		if len(mesh["vertices"].([][]float64)) < 20 || len(mesh["triangles"].([][]int)) < 24 {
			t.Fatal("Agent 使用过度简化的模型冒充目标")
		}
		bounds, _ := cloudAgentPrevisMeshBounds(mesh)
		max, _ := cloudAgentPrevisNumberVec3(bounds["max"])
		if max[1] != 4 {
			t.Fatalf("初始模型高度不正确: %v", max)
		}
		originalID, transform := stringValue(object["id"]), creationHash(object["transform"])
		update := fmt.Sprintf("继续编辑已有场景 %s 的网格 %s：先定向精读现有完整拓扑，再只把尖顶顶点从4米提高到5米，底座与支架不动；把所有红色面改为蓝色 #2266ff，保持其他面的灰色和已有对象ID、位置、旋转、缩放。必须更新原对象，不能删除或另建替代。完成后再次读回核对。不需要预览或提问。", scene["id"], originalID)
		runAgent(t, id, update, "auto", created.ID, "")
		doc = mustCreationDocument(t, canvasPayload(id))
		afterScene, _ := findPrevisScene(creationMaps(doc["previsScenes"]), stringValue(scene["id"]))
		after := cloudAgentPrevisFindByID(creationMaps(afterScene["objects"]), originalID)
		if after == nil || creationHash(after["transform"]) != transform {
			t.Fatal("修改丢失对象身份或变换")
		}
		mesh, err = cloudAgentPrevisNormalizeMesh(after["mesh"])
		if err != nil {
			t.Fatal(err)
		}
		bounds, _ = cloudAgentPrevisMeshBounds(mesh)
		max, _ = cloudAgentPrevisNumberVec3(bounds["max"])
		if max[1] != 5 {
			t.Fatalf("修改后高度不正确: %v", max)
		}
		blue := false
		for _, color := range mesh["colors"].([]string) {
			blue = blue || color == "#2266ff"
		}
		if !blue {
			t.Fatal("未修改指定面颜色")
		}
	})

	t.Run("model-repairs-original-errors", func(t *testing.T) {
		id := "live-repair-canvas"
		newCanvas(id)
		result := runAgent(t, id, "这是本地故障注入和自动纠错验收，请真实执行错误调用以验证反馈，不能只在文字里模拟。先创建空白场景并读取快照，然后按顺序：1. 用object_add kind=sculpt创建id=repair-spire，配方只有一个部件id=spire、shape=cone、size=[1,2,1]，故意触发未知形状错误；收到真实错误后，根据提示修正为合法配方并成功创建。2. 用object_add kind=prop propId=car id=repair-car params={width:2}故意触发没有可调参数；收到错误后选用支持width的参数化轿车并成功创建。两个错误必须分别真实发生且各自修正成功，不要原样重复错误。无需预览、付费生成、提问或其他建模。最后读回两件对象。", "auto", "", "")
		failed := 0
		for _, event := range result.Events {
			if event.Type == "tool_failed" && stringValue(event.Payload["toolName"]) == "previs_apply_patch" {
				failed++
			}
		}
		if failed < 2 {
			t.Fatalf("未真实触发两个原始错误：%d", failed)
		}
		doc := mustCreationDocument(t, canvasPayload(id))
		sculpt, car := false, false
		for _, scene := range creationMaps(doc["previsScenes"]) {
			for _, object := range creationMaps(scene["objects"]) {
				sculpt = sculpt || stringValue(object["kind"]) == "sculpt"
				car = car || stringValue(object["propId"]) == "shotloom-vehicle-sedan"
			}
		}
		if !sculpt || !car {
			t.Fatal("纠错后未保存合法 Sculpt 和参数化轿车")
		}
	})
	t.Run("manual-mesh-round-trip", func(t *testing.T) {
		input := os.Getenv("CANVAS_AGENT_LIVE_MESH_INPUT")
		if input == "" {
			t.Skip("提供真实浏览器手动编辑后保存的画布 JSON，验证 Agent 继续修改")
		}
		raw, err := os.ReadFile(input)
		if err != nil {
			t.Fatal(err)
		}
		doc := mustCreationDocument(t, string(raw))
		var scene, object map[string]any
		for _, candidate := range creationMaps(doc["previsScenes"]) {
			for _, item := range creationMaps(candidate["objects"]) {
				if stringValue(item["kind"]) == "mesh" {
					scene, object = candidate, item
				}
			}
		}
		if object == nil {
			t.Fatal("手动编辑结果没有可继续修改的网格")
		}
		before, err := cloudAgentPrevisNormalizeMesh(object["mesh"])
		if err != nil {
			t.Fatal(err)
		}
		vertices := before["vertices"].([][]float64)
		top := 0
		for i, v := range vertices {
			if v[1] > vertices[top][1] {
				top = i
			}
		}
		id, objectID := "live-manual-mesh-canvas", stringValue(object["id"])
		newCanvas(id)
		if err := db.Model(&model.CanvasProject{}).Where("id = ?", id).Update("payload_json", string(raw)).Error; err != nil {
			t.Fatal(err)
		}
		prompt := fmt.Sprintf("这是用户在浏览器中手动编辑并保存后的网格，请继续精确修改。场景%s，对象%s。先定向精读完整拓扑，只把顶点索引%d的y从%g改为%g，x和z不变。全部其他顶点、三角面索引、面颜色、对象ID和变换必须完全不变。必须更新原对象的mesh，不能改transform、删除或创建替代。完成后读回核对，无需预览或提问。", scene["id"], objectID, top, vertices[top][1], vertices[top][1]+1)
		runAgent(t, id, prompt, "auto", "", "")
		afterDoc := mustCreationDocument(t, canvasPayload(id))
		afterScene, _ := findPrevisScene(creationMaps(afterDoc["previsScenes"]), stringValue(scene["id"]))
		afterObject := cloudAgentPrevisFindByID(creationMaps(afterScene["objects"]), objectID)
		if afterObject == nil || creationHash(afterObject["transform"]) != creationHash(object["transform"]) {
			t.Fatal("手动编辑往返丢失对象身份或变换")
		}
		vertices[top][1]++
		if creationHash(afterObject["mesh"]) != creationHash(before) {
			t.Fatal("Agent 未精确保留手动编辑后的其他拓扑与颜色")
		}
	})

	t.Run("approval-and-rejection", func(t *testing.T) {
		id := "live-approval-canvas"
		newCanvas(id)
		approved := runAgent(t, id, "本地审批验收：新建空白场景，并自己用三角网格创建一个高2米的五顶点方锥，名为审批网格。读取规则与mesh说明，按真实审批流程操作；每次审批后继续直至保存并读回。无需预览或提问。", "request_approval", "", "approve")
		before := canvasPayload(id)
		runAgent(t, id, "本地拒绝审批验收：尝试把已有审批网格名称修改为拒绝后不应保存，使用object_update。此操作将被拒绝，收到拒绝后只报告未修改并结束，不能重试、另建对象或改写其他内容。无需预览或提问。", "request_approval", approved.ID, "reject")
		if canvasPayload(id) != before {
			t.Fatal("拒绝审批后画布发生变化")
		}
	})
}
