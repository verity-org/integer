package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/verity-org/integer/internal/config"
)

func TestApplyType(t *testing.T) {
	tests := []struct {
		name     string
		baseTags []string
		typeName string
		want     []string
	}{
		{
			name:     "default type returns base tags unchanged",
			baseTags: []string{"22", "latest"},
			typeName: "default",
			want:     []string{"22", "latest"},
		},
		{
			name:     "dev type appends -dev suffix",
			baseTags: []string{"22", "latest"},
			typeName: "dev",
			want:     []string{"22-dev", "latest-dev"},
		},
		{
			name:     "fips type appends -fips suffix",
			baseTags: []string{"3.12"},
			typeName: "fips",
			want:     []string{"3.12-fips"},
		},
		{
			name:     "empty base tags returns empty slice",
			baseTags: []string{},
			typeName: "default",
			want:     []string{},
		},
		{
			name:     "empty base tags with non-default type returns empty slice",
			baseTags: []string{},
			typeName: "dev",
			want:     []string{},
		},
		{
			name:     "does not mutate original slice",
			baseTags: []string{"1.0"},
			typeName: "dev",
			want:     []string{"1.0-dev"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := make([]string, len(tt.baseTags))
			copy(original, tt.baseTags)

			got := config.ApplyType(tt.baseTags, tt.typeName)

			assert.Equal(t, tt.want, got)
			// Verify the input slice was not mutated
			assert.Equal(t, original, tt.baseTags)
		})
	}
}
