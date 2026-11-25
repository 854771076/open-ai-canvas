package protocol

import (
	"context"
	"testing"
)

func TestNewAPIGatewayTasks(t *testing.T) {
	for _, capability := range []Capability{CapabilityImage, CapabilityVideo} {
		t.Run(string(capability), func(t *testing.T) {
			a := officialPackageAdapter(t, "../custom-plugins/newapi-gateway-tasks.yingce-plugin", "newapi-runninghub-"+string(capability))
			request := GenerationRequest{Capability: capability, Model: "krea2-normal-v1", Prompt: "a circle", AspectRatio: "1:1", Resolution: "2K", Duration: 5, Images: []MediaReference{{URL: "https://assets.example/reference.png"}}}
			spec, err := a.BuildCreate(context.Background(), RequestContext{Request: request})
			if err != nil {
				t.Fatal(err)
			}
			body := manifestTestBody(t, spec)
			if spec.Path != "/v1/tasks/runninghub" || spec.ContentType != "multipart/form-data" || len(spec.Files) != 1 || body["model"] != request.Model || body["aspect_ratio"] != "1:1" || body["resolution"] != "2K" {
				t.Fatalf("invalid submit: %#v %#v", spec, body)
			}
			state, err := a.ParseCreate(context.Background(), []byte(`{"task_id":"task_1","status":"SUBMITTED"}`))
			if err != nil || state.TaskID != "task_1" || state.Status == StatusSucceeded {
				t.Fatalf("invalid create: %#v %v", state, err)
			}
			pollContext := PollContext{TaskID: "task_1", Request: request}
			for _, status := range []string{"IN_PROGRESS", "SUCCESS", "FAILURE"} {
				state, err := a.ParsePoll(context.Background(), pollContext, []byte(`{"task_id":"task_1","status":"`+status+`","fail_reason":"upstream rejected"}`))
				if err != nil {
					t.Fatal(err)
				}
				if status == "SUCCESS" && state.Status != StatusSucceeded || status == "FAILURE" && (state.Status != StatusFailed || state.Message != "upstream rejected") || status == "IN_PROGRESS" && state.Status == StatusSucceeded {
					t.Fatalf("invalid status: %#v", state)
				}
			}
			spec, err = a.(ResultAdapter).BuildResult(context.Background(), pollContext)
			if err != nil || spec.Path != "/v1/tasks/task_1/artifacts" {
				t.Fatalf("invalid result request: %#v %v", spec, err)
			}
			state2, err := a.ParsePoll(context.Background(), pollContext, []byte(`{"task_id":"task_1","artifacts":[{"type":"`+string(capability)+`","content_url":"/v1/tasks/task_1/artifacts/result/content?access=SIGNED","mime_type":"`+string(capability)+`/test"}]}`))
			if err != nil || state2.Status != StatusSucceeded || state2.Result == nil {
				t.Fatalf("invalid artifacts: %#v %v", state2, err)
			}
			media := state2.Result.Images
			if capability == CapabilityVideo {
				media = state2.Result.Videos
			}
			if len(media) != 1 || !media[0].Ephemeral || media[0].URL != "/v1/tasks/task_1/artifacts/result/content?access=SIGNED" {
				t.Fatalf("invalid media: %#v", media)
			}
			if capability == CapabilityVideo {
				request.Images = make([]MediaReference, 5)
				if _, err := a.BuildCreate(context.Background(), RequestContext{Request: request}); err == nil {
					t.Fatal("accepted more reference uploads than the gateway supports")
				}
			}
		})
	}
}
