package app

import (
	"strings"
	"testing"
)

func TestCloudAgentPrevisShotloomCatalog(t *testing.T) {
	count := 0
	for _, prop := range cloudAgentPrevisPropCatalog {
		if !strings.HasPrefix(prop.ID, "shotloom-") {
			continue
		}
		count++
		if cloudAgentPrevisFootprints[prop.ID] <= 0 {
			t.Fatalf("缺少占地尺寸：%s", prop.ID)
		}
		params, err := cloudAgentPrevisNormalizeParams(prop.ID, nil, nil)
		if err != nil || len(params) == 0 {
			t.Fatalf("默认参数无效 %s: %v", prop.ID, err)
		}
		if _, err := cloudAgentPrevisNormalizeParams(prop.ID, nil, map[string]any{"unsupported": 1}); err == nil {
			t.Fatalf("未知参数未拒绝：%s", prop.ID)
		}
	}
	if count != 136 {
		t.Fatalf("预设数量 = %d，期望 136", count)
	}
}

func TestCloudAgentPrevisShotloomNumericChoices(t *testing.T) {
	params, err := cloudAgentPrevisNormalizeParams("shotloom-theme-ring", nil, map[string]any{"ropes": "0"})
	if err != nil || params["ropes"] != "0" {
		t.Fatalf("数字选项应使用字符串枚举：%v %v", params, err)
	}
	for _, value := range []any{"2", 1, "invalid"} {
		if _, err := cloudAgentPrevisNormalizeParams("shotloom-theme-ring", nil, map[string]any{"ropes": value}); err == nil {
			t.Fatalf("非法枚举未拒绝：%v", value)
		}
	}
	if _, err := cloudAgentPrevisNormalizeParams("shotloom-shape-box", nil, map[string]any{"width": 501}); err == nil {
		t.Fatal("越界尺寸未拒绝")
	}
}
