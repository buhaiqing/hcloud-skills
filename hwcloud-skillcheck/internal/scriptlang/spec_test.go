package scriptlang

import "testing"

func TestSpec_LongestPrefixWins(t *testing.T) {
	spec := &Spec{
		Entries: []Entry{
			{Match: "ecs list", Response: `[{"id":"a"}]`, ExitCode: 0},
			{Match: "ecs list-servers", Response: `[]`, ExitCode: 0},
		},
	}
	got, ok := spec.Match("ecs list-servers --region cn-north-4")
	if !ok || got.Response != "[]" {
		t.Errorf("got %+v ok=%v, want list-servers response", got, ok)
	}
}
