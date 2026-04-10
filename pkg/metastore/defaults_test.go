package metastore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
)

func Test_defaultIfNil(t *testing.T) {
	t.Run("Nil Bool", func(t *testing.T) {
		var b *bool
		require.Zero(t, defaultIfNil(b), "Expected false for nil bool")
	})

	t.Run("Non-nil Bool", func(t *testing.T) {
		b := true
		require.True(t, defaultIfNil(&b), "Expected true for non-nil bool")
	})

	t.Run("Nil String", func(t *testing.T) {
		var s *string
		require.Zero(t, defaultIfNil(s), "Expected empty string for nil string")
	})

	t.Run("Non-nil String", func(t *testing.T) {
		s := "Hello"
		require.Equal(t, "Hello", defaultIfNil(&s), "Expected 'Hello' for non-nil string")
	})

	t.Run("Nil Int", func(t *testing.T) {
		var i *int
		require.Zero(t, defaultIfNil(i), "Expected empty string for nil string")
	})

	t.Run("Non-nil Int", func(t *testing.T) {
		i := 64
		require.Equal(t, 64, defaultIfNil(&i), "Expected '64' for non-nil int")
	})

	t.Run("Run len() method to a nil array", func(t *testing.T) {
		var cmpSpec models.ComponentSpec
		got := defaultIfNil(cmpSpec.ExposedInterfaces)
		l := len(got)
		require.Equal(t, 0, l)
	})

}
