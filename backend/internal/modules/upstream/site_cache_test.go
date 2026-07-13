package upstream

import (
	"encoding/json"
	"testing"
)

func TestSitePayloadRoundTripPreservesSettings(t *testing.T) {
	threshold := 10.5
	site := &Site{
		ID:             "site-1",
		UserID:         "user-1",
		AdminAccountID: "account-1",
		Settings: SiteSettings{
			BalanceThreshold: &threshold,
		},
	}

	raw, err := json.Marshal(toPayload(site))
	if err != nil {
		t.Fatalf("marshal site payload: %v", err)
	}

	var payload sitePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal site payload: %v", err)
	}

	restored := fromPayload(payload)
	if restored.Settings.BalanceThreshold == nil {
		t.Fatal("balance threshold was lost during cache serialization")
	}
	if got := *restored.Settings.BalanceThreshold; got != threshold {
		t.Errorf("balance threshold = %v, want %v", got, threshold)
	}
}
