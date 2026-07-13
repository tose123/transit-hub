package httpserver

import (
	"context"
	"errors"
	"strings"
	"testing"

	"transithub/backend/internal/modules/settings"
	"transithub/backend/internal/modules/upstream"
)

type fakeMultiplierAlertService struct {
	keys     []upstream.Sub2APIKeyItem
	keyErr   error
	keyCalls int
	messages []string
	userID   string
	siteID   string
}

func (f *fakeMultiplierAlertService) ListUpstreamKeys(_ context.Context, userID string, siteID string) ([]upstream.Sub2APIKeyItem, error) {
	f.keyCalls++
	f.userID = userID
	f.siteID = siteID
	return f.keys, f.keyErr
}

func (f *fakeMultiplierAlertService) SendToBots(_ context.Context, _ string, _ []string, message string) {
	f.messages = append(f.messages, message)
}

func TestCheckMultiplierChangesIncludesKeyBindings(t *testing.T) {
	oldRateA, newRateA := 1.0, 1.2
	oldRateB, newRateB := 2.0, 1.8
	service := &fakeMultiplierAlertService{
		keys: []upstream.Sub2APIKeyItem{
			{ID: "key-1", Name: "生产 Key", GroupID: "group-a"},
			{ID: "key-2", Name: "备用 Key", GroupName: "分组 A"},
		},
	}
	strategy := settings.StrategySettings{
		EnableMultiplierAlert:  true,
		MultiplierNotifyBotIDs: []string{"bot-1"},
	}

	checkMultiplierChanges(context.Background(), service, service, strategy, "user-1", "site-1", "上游站点", upstream.Metrics{
		Groups: []upstream.GroupInfo{
			{ID: "group-a", Name: "分组 A", Multiplier: &oldRateA},
			{ID: "group-b", Name: "分组 B", Multiplier: &oldRateB},
		},
	}, upstream.Metrics{
		Groups: []upstream.GroupInfo{
			{ID: "group-a", Name: "分组 A", Multiplier: &newRateA},
			{ID: "group-b", Name: "分组 B", Multiplier: &newRateB},
		},
	})

	if service.keyCalls != 1 {
		t.Fatalf("Key 列表查询次数 = %d，期望 1", service.keyCalls)
	}
	if service.userID != "user-1" || service.siteID != "site-1" {
		t.Fatalf("Key 列表查询参数 = (%q, %q)", service.userID, service.siteID)
	}
	if len(service.messages) != 2 {
		t.Fatalf("提醒数量 = %d，期望 2", len(service.messages))
	}
	if !strings.Contains(service.messages[0], "是否绑定 Key：是；Key 名称：生产 Key、备用 Key") {
		t.Fatalf("已绑定提醒未包含 Key 名称：%q", service.messages[0])
	}
	if !strings.Contains(service.messages[1], "是否绑定 Key：否；Key 名称：无") {
		t.Fatalf("未绑定提醒内容错误：%q", service.messages[1])
	}
}

func TestCheckMultiplierChangesKeepsAlertWhenKeyListFails(t *testing.T) {
	oldRate, newRate := 1.0, 1.5
	service := &fakeMultiplierAlertService{keyErr: errors.New("上游不可用")}
	strategy := settings.StrategySettings{
		EnableMultiplierAlert:  true,
		MultiplierNotifyBotIDs: []string{"bot-1"},
	}

	checkMultiplierChanges(context.Background(), service, service, strategy, "user-1", "site-1", "上游站点", upstream.Metrics{
		Groups: []upstream.GroupInfo{{ID: "group-a", Name: "分组 A", Multiplier: &oldRate}},
	}, upstream.Metrics{
		Groups: []upstream.GroupInfo{{ID: "group-a", Name: "分组 A", Multiplier: &newRate}},
	})

	if service.keyCalls != 1 || len(service.messages) != 1 {
		t.Fatalf("Key 查询次数 = %d，提醒数量 = %d", service.keyCalls, len(service.messages))
	}
	if !strings.Contains(service.messages[0], "是否绑定 Key：未知；Key 名称：获取失败") {
		t.Fatalf("Key 查询失败时提醒内容错误：%q", service.messages[0])
	}
}
