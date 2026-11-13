package discord

import (
	"encoding/json"
	"testing"
)

func TestRoleColors(t *testing.T) {
	var r Role
	if err := json.Unmarshal([]byte(`{"id":"1","colors":{"primary_color":255,"secondary_color":null,"tertiary_color":null},"flags":1}`), &r); err != nil {
		t.Fatal(err)
	}
	want := RoleColors{Primary: 255, Secondary: NullColor, Tertiary: NullColor}
	if r.Colors != want || r.Flags != RoleInPrompt {
		t.Fatalf("got %+v, flags %d", r.Colors, r.Flags)
	}

	b, err := json.Marshal(r.Colors)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"primary_color":255,"secondary_color":null,"tertiary_color":null}` {
		t.Fatalf("got %s", b)
	}
}
