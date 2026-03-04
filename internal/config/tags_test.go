package config_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verity-org/integer/internal/config"
)

var errStop = errors.New("stop")

func TestForEachType(t *testing.T) {
	t.Run("iterates all types with computed tags", func(t *testing.T) {
		v := config.VersionDef{
			Version: "22",
			Tags:    []string{"22", "latest"},
			Types:   []string{"default", "dev", "fips"},
		}
		type result struct {
			typeName string
			tags     []string
		}
		var got []result
		require.NoError(t, config.ForEachType(&v, func(typeName string, tags []string) error {
			got = append(got, result{typeName, tags})
			return nil
		}))
		require.Len(t, got, 3)
		assert.Equal(t, "default", got[0].typeName)
		assert.Equal(t, []string{"22", "latest"}, got[0].tags)
		assert.Equal(t, "dev", got[1].typeName)
		assert.Equal(t, []string{"22-dev", "latest-dev"}, got[1].tags)
		assert.Equal(t, "fips", got[2].typeName)
	})

	t.Run("skips empty tags", func(t *testing.T) {
		v := config.VersionDef{Tags: []string{}, Types: []string{"default", "dev"}}
		var count int
		require.NoError(t, config.ForEachType(&v, func(_ string, _ []string) error {
			count++
			return nil
		}))
		assert.Equal(t, 0, count)
	})

	t.Run("propagates callback error and stops", func(t *testing.T) {
		v := config.VersionDef{Tags: []string{"1"}, Types: []string{"default", "dev", "fips"}}
		var count int
		err := config.ForEachType(&v, func(_ string, _ []string) error {
			count++
			return errStop
		})
		require.Error(t, err)
		assert.Equal(t, 1, count) // stopped after first call
	})
}

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
