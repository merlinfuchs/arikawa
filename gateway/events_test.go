package gateway

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/ws"
)

func TestRequestGuildMembersCommand(t *testing.T) {
	assert := func(cmd Event, data map[string]any) {
		cmdBytes, err := json.Marshal(cmd)
		if err != nil {
			t.Fatal("failed to marshal command:", err)
		}

		var cmdMap map[string]any
		if err := json.Unmarshal(cmdBytes, &cmdMap); err != nil {
			t.Fatal("failed to unmarshal command:", err)
		}

		if !reflect.DeepEqual(cmdMap, data) {
			t.Fatalf("mismatched command, got %#v", cmdMap)
		}
	}

	t.Run("userIDs", func(t *testing.T) {
		cmd := RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{123},
			UserIDs:  []discord.UserID{456},
		}

		assert(&cmd, map[string]any{
			"guild_id":  []any{"123"},
			"user_ids":  []any{"456"},
			"presences": false,
		})
	})

	t.Run("query_empty", func(t *testing.T) {
		cmd := RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{123},
			Query:    option.NewString(""),
		}

		assert(&cmd, map[string]any{
			"guild_id":  []any{"123"},
			"query":     "",
			"limit":     float64(0),
			"presences": false,
		})
	})

	t.Run("query_nonempty", func(t *testing.T) {
		cmd := RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{123},
			Query:    option.NewString("abc"),
		}

		assert(&cmd, map[string]any{
			"guild_id":  []any{"123"},
			"query":     "abc",
			"limit":     float64(0),
			"presences": false,
		})
	})

	t.Run("both", func(t *testing.T) {
		cmd := RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{123},
			UserIDs:  []discord.UserID{456},
			Query:    option.NewString("abc"),
		}

		// Gateway should never be touched when Marshal fails, so we can just
		// create a zero-value.
		var gateway ws.Gateway

		err := gateway.Send(context.Background(), &cmd)
		if err == nil || !strings.Contains(err.Error(), "neither UserIDs nor Query can be filled") {
			t.Fatal("unexpected error:", err)
		}
	})
}

func TestAutoModerationActionExecutionEvent(t *testing.T) {
	const payload = `{
		"guild_id": "1",
		"action": {"type": 1, "metadata": {"custom_message": "No invites"}},
		"rule_id": "2",
		"rule_trigger_type": 1,
		"user_id": "3",
		"channel_id": "4",
		"alert_system_message_id": "5",
		"content": "join discord.gg/abc",
		"matched_keyword": "discord.gg",
		"matched_content": "discord.gg"
	}`

	ev := OpUnmarshalers.Lookup(dispatchOp, "AUTO_MODERATION_ACTION_EXECUTION")
	if ev == nil {
		t.Fatal("AUTO_MODERATION_ACTION_EXECUTION isn't registered")
	}
	got, ok := ev().(*AutoModerationActionExecutionEvent)
	if !ok {
		t.Fatalf("registered as %T", ev())
	}
	if err := json.Unmarshal([]byte(payload), got); err != nil {
		t.Fatal("failed to unmarshal:", err)
	}

	want := AutoModerationActionExecutionEvent{
		GuildID: 1,
		Action: discord.AutoModerationAction{
			Type:     discord.AutoModerationBlockMessage,
			Metadata: discord.AutoModerationActionMetadata{CustomMessage: "No invites"},
		},
		RuleID:               2,
		RuleTriggerType:      discord.AutoModerationKeyword,
		UserID:               3,
		ChannelID:            4,
		AlertSystemMessageID: 5,
		Content:              "join discord.gg/abc",
		MatchedKeyword:       "discord.gg",
		MatchedContent:       "discord.gg",
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("got %#v", *got)
	}

	if EventIntents["AUTO_MODERATION_ACTION_EXECUTION"] != IntentAutoModerationExecution {
		t.Fatal("AUTO_MODERATION_ACTION_EXECUTION doesn't need IntentAutoModerationExecution")
	}
}

func TestAutoModerationRuleTriggerType(t *testing.T) {
	var rule AutoModerationRuleCreateEvent
	if err := json.Unmarshal([]byte(`{"id": "1", "name": "Spam", "trigger_type": 3}`), &rule); err != nil {
		t.Fatal("failed to unmarshal:", err)
	}
	if rule.TriggerType != discord.AutoModerationSpam {
		t.Fatalf("got trigger type %d", rule.TriggerType)
	}
}
