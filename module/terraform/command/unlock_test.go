package command

import (
	"strings"
	"testing"

	"github.com/fabienChaillou/terraform-commander/terraform"
)

func TestUnlockValidate(t *testing.T) {
	cmd := &UnlockCommand{}

	tests := []struct {
		name      string
		req       *terraform.Request
		wantErrs  int
		errFields []string
	}{
		{
			name:      "missing lock-id",
			req:       &terraform.Request{Action: "unlock", Args: map[string]interface{}{}},
			wantErrs:  1,
			errFields: []string{"lock-id"},
		},
		{
			name:      "blank lock-id",
			req:       &terraform.Request{Action: "unlock", Args: map[string]interface{}{"lock-id": "   "}},
			wantErrs:  1,
			errFields: []string{"lock-id"},
		},
		{
			name:     "valid lock-id",
			req:      &terraform.Request{Action: "unlock", Args: map[string]interface{}{"lock-id": "1234abcd"}},
			wantErrs: 0,
		},
		{
			name:     "lock-id + force",
			req:      &terraform.Request{Action: "unlock", Args: map[string]interface{}{"lock-id": "1234abcd", "force": true}},
			wantErrs: 0,
		},
		{
			name:      "unknown flag",
			req:       &terraform.Request{Action: "unlock", Args: map[string]interface{}{"lock-id": "x", "noisy": true}},
			wantErrs:  1,
			errFields: []string{"noisy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrors(t, cmd.Validate(tt.req), tt.wantErrs, tt.errFields)
		})
	}
}

func TestUnlockBuildArgs_UsesForceUnlockVerb(t *testing.T) {
	cmd := &UnlockCommand{}
	args := cmd.BuildArgs(&terraform.Request{Args: map[string]interface{}{"lock-id": "1234abcd"}})
	if args[0] != "force-unlock" {
		t.Errorf("first arg should be 'force-unlock' (not 'unlock'), got %v", args)
	}
	if args[len(args)-1] != "1234abcd" {
		t.Errorf("lock-id should be the trailing positional arg, got %v", args)
	}
}

func TestUnlockBuildArgs_WithForce(t *testing.T) {
	cmd := &UnlockCommand{}
	args := cmd.BuildArgs(&terraform.Request{Args: map[string]interface{}{"lock-id": "x", "force": true}})

	forceIdx, idIdx := -1, -1
	for i, a := range args {
		if a == "-force" {
			forceIdx = i
		}
		if a == "x" {
			idIdx = i
		}
	}
	if forceIdx == -1 || idIdx == -1 {
		t.Fatalf("expected -force and lock-id in %v", args)
	}
	if forceIdx >= idIdx {
		t.Errorf("-force (idx %d) should precede lock-id (idx %d)", forceIdx, idIdx)
	}
}

func TestUnlockHelp_MentionsForceUnlock(t *testing.T) {
	if !strings.Contains((&UnlockCommand{}).Help(), "force-unlock") {
		t.Error("UnlockCommand.Help() should mention 'force-unlock'")
	}
}
