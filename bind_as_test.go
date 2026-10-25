package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockDatabase() *mockDatabase {
	return &mockDatabase{}
}

func newUserService(db Database) *userServiceImpl {
	return &userServiceImpl{db: db}
}

func TestContainer_BindAs(t *testing.T) {
	t.Run("bind concrete constructor as interface", func(t *testing.T) {
		container := New()

		err := container.Bind(newMockDatabase, As[Database]())
		require.NoError(t, err)

		var db Database
		err = container.Resolve(&db)
		require.NoError(t, err)
		assert.IsType(t, &mockDatabase{}, db)

		// The concrete type itself is not bound
		var concrete *mockDatabase
		assert.Error(t, container.Resolve(&concrete))
	})

	t.Run("interface binding is injected into dependents", func(t *testing.T) {
		container := New()

		require.NoError(t, container.Bind(newMockDatabase, As[Database]()))
		require.NoError(t, container.Bind(newUserService, As[UserService]()))

		var svc UserService
		require.NoError(t, container.Resolve(&svc))

		var db Database
		require.NoError(t, container.Resolve(&db))
		assert.Same(t, db, svc.(*userServiceImpl).db)
	})

	t.Run("works with named, transient and eager options", func(t *testing.T) {
		container := New()

		require.NoError(t, container.BindNamedTransient("primary", newMockDatabase, As[Database]()))
		require.NoError(t, container.Bind(newMockDatabase, As[Database](), WithName("eager"), WithEager()))

		var db1, db2 Database
		require.NoError(t, container.ResolveNamed(&db1, "primary"))
		require.NoError(t, container.ResolveNamed(&db2, "primary"))
		assert.NotSame(t, db1, db2)

		var all []Database
		require.NoError(t, container.ResolveAll(&all))
		assert.Len(t, all, 2)
	})

	t.Run("error when return type does not implement interface", func(t *testing.T) {
		container := New()

		err := container.Bind(func() *loggerImpl { return &loggerImpl{} }, As[Database]())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not assignable")
	})

	t.Run("error when resolver depends on bound type", func(t *testing.T) {
		container := New()

		err := container.Bind(func(db Database) *mockDatabase { return &mockDatabase{} }, As[Database]())
		assert.Error(t, err)
	})
}

func TestGlobal_BindAs(t *testing.T) {
	Clear()
	defer Clear()

	require.NoError(t, BindAs[Database](newMockDatabase))

	var db Database
	require.NoError(t, Resolve(&db))
	assert.IsType(t, &mockDatabase{}, db)
}
