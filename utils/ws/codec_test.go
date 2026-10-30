package ws

import (
	"context"
	"strings"
	"testing"
)

type seqTestEvent struct{ N int }

func (*seqTestEvent) Op() OpCode           { return 5 }
func (*seqTestEvent) EventType() EventType { return "" }

func TestDecodeIntoSequence(t *testing.T) {
	codec := NewCodec(NewOpUnmarshalers(func() Event { return new(seqTestEvent) }))

	tests := []struct {
		name string
		json string
		want int64
	}{
		{"main gateway", `{"op":5,"d":{"N":1},"s":12}`, 12},
		{"voice gateway", `{"op":5,"d":{"N":1},"seq":13}`, 13},
		{"none", `{"op":5,"d":{"N":1}}`, 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out := make(chan Op, 1)
			buf := NewDecodeBuffer(0)

			if err := codec.DecodeInto(context.Background(), strings.NewReader(test.json), &buf, out); err != nil {
				t.Fatal("failed to decode:", err)
			}

			op := <-out
			if _, ok := op.Data.(*seqTestEvent); !ok {
				t.Fatalf("got event %#v", op.Data)
			}
			if op.Sequence != test.want {
				t.Errorf("got sequence %d, want %d", op.Sequence, test.want)
			}
		})
	}
}
