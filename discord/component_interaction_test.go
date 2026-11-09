package discord

import (
	"reflect"
	"testing"

	"github.com/diamondburned/arikawa/v3/utils/json"
)

func TestComponentInteractionRoundTrip(t *testing.T) {
	tests := []ComponentInteraction{
		&ButtonInteraction{CustomID: "button"},
		&StringSelectInteraction{CustomID: "string", Values: []string{"a", "b"}},
		&ChannelSelectInteraction{CustomID: "channel", Values: []ChannelID{1}},
		&RoleSelectInteraction{CustomID: "role", Values: []RoleID{2}},
		&UserSelectInteraction{CustomID: "user", Values: []UserID{3}},
		&MentionableSelectInteraction{CustomID: "mentionable", Values: []Snowflake{4}},
	}

	for _, want := range tests {
		b, err := json.Marshal(&InteractionEvent{ID: 5, Data: want})
		if err != nil {
			t.Fatalf("marshal %T: %v", want, err)
		}

		var got InteractionEvent
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if !reflect.DeepEqual(got.Data, want) {
			t.Errorf("round trip %s: got %#v, want %#v", b, got.Data, want)
		}
	}
}
