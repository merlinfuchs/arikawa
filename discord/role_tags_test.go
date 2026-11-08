package discord

import (
	"testing"

	"github.com/diamondburned/arikawa/v3/utils/json"
)

func TestRoleTagsPremiumSubscriber(t *testing.T) {
	tests := []struct {
		json string
		want RoleTags
	}{
		{`{}`, RoleTags{}},
		{`{"premium_subscriber":null}`, RoleTags{PremiumSubscriber: true}},
		{`{"bot_id":"1"}`, RoleTags{BotID: 1}},
		{`{"integration_id":"2","premium_subscriber":null}`, RoleTags{IntegrationID: 2, PremiumSubscriber: true}},
	}

	for _, test := range tests {
		var got RoleTags
		if err := json.Unmarshal([]byte(test.json), &got); err != nil {
			t.Fatalf("unmarshal %s: %v", test.json, err)
		}
		if got != test.want {
			t.Errorf("unmarshal %s: got %+v, want %+v", test.json, got, test.want)
		}

		b, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("marshal %+v: %v", got, err)
		}
		var again RoleTags
		if err := json.Unmarshal(b, &again); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if again != got {
			t.Errorf("round trip of %s: got %+v from %s", test.json, again, b)
		}
	}
}

func TestRolePremiumSubscriberTag(t *testing.T) {
	var role Role
	err := json.Unmarshal([]byte(`{"id":"1","name":"Server Booster","managed":true,"tags":{"premium_subscriber":null}}`), &role)
	if err != nil {
		t.Fatal(err)
	}
	if !role.Tags.PremiumSubscriber {
		t.Error("PremiumSubscriber is false")
	}
}
