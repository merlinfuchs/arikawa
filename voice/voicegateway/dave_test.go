package voicegateway

import (
	"bytes"
	"encoding"
	"reflect"
	"testing"

	"github.com/diamondburned/arikawa/v3/utils/ws"
)

func TestUnmarshalBinaryOp(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		seq  int64
		want ws.Event
	}{
		{
			name: "external sender package",
			in:   []byte{0, 3, 25, 1, 2, 3},
			seq:  3,
			want: &DAVEMLSExternalSenderPackageEvent{ExternalSenderPackage: []byte{1, 2, 3}},
		},
		{
			name: "proposals",
			in:   []byte{1, 0, 27, 0, 9},
			seq:  256,
			want: &DAVEMLSProposalsEvent{Proposals: []byte{0, 9}},
		},
		{
			name: "announce commit transition",
			in:   []byte{0, 4, 29, 0, 7, 5, 6},
			seq:  4,
			want: &DAVEMLSAnnounceCommitTransitionEvent{TransitionID: 7, Commit: []byte{5, 6}},
		},
		{
			name: "welcome",
			in:   []byte{0, 5, 30, 1, 2, 8},
			seq:  5,
			want: &DAVEMLSWelcomeEvent{TransitionID: 258, Welcome: []byte{8}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			op, err := UnmarshalBinaryOp(bytes.NewReader(test.in))
			if err != nil {
				t.Fatal("unexpected error:", err)
			}
			if op.Sequence != test.seq {
				t.Errorf("got sequence %d, want %d", op.Sequence, test.seq)
			}
			if op.Code != test.want.Op() {
				t.Errorf("got op %d, want %d", op.Code, test.want.Op())
			}
			if !reflect.DeepEqual(op.Data, test.want) {
				t.Errorf("got %#v, want %#v", op.Data, test.want)
			}
		})
	}
}

func TestUnmarshalBinaryOpInvalid(t *testing.T) {
	for _, in := range [][]byte{{0, 1}, {0, 1, 29, 0}, {0, 1, 99, 1}} {
		if _, err := UnmarshalBinaryOp(bytes.NewReader(in)); err == nil {
			t.Errorf("expected an error for %v", in)
		}
	}
}

func TestMarshalBinaryCommand(t *testing.T) {
	tests := []struct {
		cmd  encoding.BinaryMarshaler
		want []byte
	}{
		{&DAVEMLSKeyPackageCommand{KeyPackage: []byte{1, 2}}, []byte{26, 1, 2}},
		{&DAVEMLSCommitWelcomeCommand{CommitWelcome: []byte{3}}, []byte{28, 3}},
	}

	for _, test := range tests {
		got, err := test.cmd.MarshalBinary()
		if err != nil {
			t.Fatal("failed to marshal:", err)
		}
		if !bytes.Equal(got, test.want) {
			t.Errorf("got %v, want %v", got, test.want)
		}
	}
}
