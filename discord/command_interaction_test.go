package discord

import (
	"testing"

	"github.com/diamondburned/arikawa/v3/utils/json"
)

func TestCommandInteractionType(t *testing.T) {
	b := []byte(`{"id":"1","type":2,"data":{"id":"2","name":"Report","type":3,"target_id":"3"}}`)

	var ev InteractionEvent
	if err := json.Unmarshal(b, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cmd, ok := ev.Data.(*CommandInteraction)
	if !ok {
		t.Fatalf("data is %T, want *CommandInteraction", ev.Data)
	}
	if cmd.Type != MessageCommand {
		t.Errorf("type is %v, want %v", cmd.Type, MessageCommand)
	}
}
