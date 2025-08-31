package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)



func TestGlobalFunctions(t *testing.T) {
	t.Run("bind and resolve simple", func(t *testing.T) {
		Clear() // Ensure clean state
		err := Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		var db Database
		err = Resolve(&db)
		assert.NoError(t, err)
		assert.NotNil(t, db)
		Clear() // Clean up
	})

	t.Run("singleton behavior with global functions", func(t *testing.T) {
		Clear()
		err := Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		var db1, db2 Database
		err = Resolve(&db1)
		require.NoError(t, err)
		err = Resolve(&db2)
		require.NoError(t, err)

		assert.Same(t, db1, db2)
		Clear()
	})

	t.Run("transient behavior with global functions", func(t *testing.T) {
		Clear()
		err := BindTransient(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		var db1, db2 Database
		err = Resolve(&db1)
		require.NoError(t, err)
		err = Resolve(&db2)
		require.NoError(t, err)

		assert.NotSame(t, db1, db2)
		Clear()
	})

	t.Run("named binding with global functions", func(t *testing.T) {
		Clear()
		err := BindNamed("primary", func() Database {
			return &mockDatabase{connected: true}
		})
		require.NoError(t, err)

		var db Database
		err = ResolveNamed(&db, "primary")
		require.NoError(t, err)
		assert.NotNil(t, db)
		assert.True(t, db.(*mockDatabase).connected)
		Clear()
	})

	t.Run("clear global container", func(t *testing.T) {
		Clear()
		err := Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		Clear()

		var db Database
		err = Resolve(&db)
		assert.Error(t, err)
	})
}
