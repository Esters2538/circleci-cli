package runner

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/CircleCI-Public/circleci-cli/api/runner"
)

func TestAgentConfig_WriteYaml(t *testing.T) {
	token := runner.Token{
		ID:            "da73786c-ebbc-4c07-849a-5590f7eef509",
		Token:         "1a34e5519976717fb808ad8900cadbecc686facee3f9ca56c5ba1ad30e50cab7e5fa328409065c64",
		ResourceClass: "the-namespace/the-resource-class",
		Nickname:      "the-nickname",
		CreatedAt:     time.Date(2020, 03, 04, 16, 13, 53, 00, time.UTC),
	}

	tests := []struct {
		platform string
		wantErr  string
	}{
		{
			platform: "minimal",
		},
		{
			platform: "linux",
		},
		{
			platform: "macos",
		},
		{
			platform: "unknown",
			wantErr:  "unknown platform",
		},
	}
	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			b := bytes.Buffer{}
			err := generateConfig(token, tt.platform, &b)
			if tt.wantErr == "" {
				assert.NilError(t, err)
				golden.Assert(t, b.String(), fmt.Sprintf("expected-config-%s.yaml", tt.platform))
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
