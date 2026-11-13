package discord

import (
	"encoding/json"
	"testing"
)

func TestUserPrimaryGuild(t *testing.T) {
	var u User
	if err := json.Unmarshal([]byte(`{"id":"1","primary_guild":{"identity_guild_id":"2","identity_enabled":true,"tag":"KITE","badge":"abc"}}`), &u); err != nil {
		t.Fatal(err)
	}
	if u.PrimaryGuild == nil || u.PrimaryGuild.IdentityGuildID != 2 || u.PrimaryGuild.Tag != "KITE" || !*u.PrimaryGuild.IdentityEnabled {
		t.Fatalf("got %+v", u.PrimaryGuild)
	}
}
