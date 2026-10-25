package di

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cycleA interface{ A() }
type cycleB interface{ B() }
type cycleC interface{ C() }

type cycleImpl struct{}

func (cycleImpl) A() {}
func (cycleImpl) B() {}
func (cycleImpl) C() {}

// withTimeout fails the test instead of hanging if f deadlocks.
func withTimeout(t *testing.T, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out, resolution is probably deadlocked")
	}
}

func TestContainer_NilResult(t *testing.T) {
	t.Run("nil interface returns error instead of panicking", func(t *testing.T) {
		container := New()
		require.NoError(t, container.Bind(func() Database { return nil }))

		var db Database
		assert.NotPanics(t, func() {
			err := container.Resolve(&db)
			assert.EqualError(t, err, "resolver for di.Database returned nil")
		})
		assert.Nil(t, db)
	})

	t.Run("nil interface dependency fails the dependent", func(t *testing.T) {
		container := New()
		require.NoError(t, container.Bind(func() Database { return nil }))
		require.NoError(t, container.Bind(func(db Database) UserService {
			return &userServiceImpl{db: db}
		}))

		var svc UserService
		assert.EqualError(t, container.Resolve(&svc), "resolver for di.Database returned nil")
	})

	t.Run("eager binding returning nil fails the bind", func(t *testing.T) {
		container := New()
		err := container.Bind(func() Database { return nil }, WithEager())
		assert.EqualError(t, err, "resolver for di.Database returned nil")

		var db Database
		assert.Error(t, container.Resolve(&db))
	})

	t.Run("resolver error takes precedence over nil value", func(t *testing.T) {
		container := New()
		require.NoError(t, container.Bind(func() (Database, error) {
			return nil, errors.New("connection refused")
		}))

		var db Database
		assert.EqualError(t, container.Resolve(&db), "connection refused")
	})

	t.Run("nil pointer of a concrete type is still a valid result", func(t *testing.T) {
		container := New()
		require.NoError(t, container.Bind(func() *mockDatabase { return nil }))

		var db *mockDatabase
		assert.NoError(t, container.Resolve(&db))
		assert.Nil(t, db)
	})
}

func TestContainer_CircularDependency(t *testing.T) {
	bindAB := func(c *Container, options ...BindOption) {
		require.NoError(t, c.Bind(func(cycleB) cycleA { return cycleImpl{} }, options...))
		require.NoError(t, c.Bind(func(cycleA) cycleB { return cycleImpl{} }, options...))
	}

	t.Run("singletons return error instead of deadlocking", func(t *testing.T) {
		container := New()
		bindAB(container)

		withTimeout(t, func() {
			var a cycleA
			err := container.Resolve(&a)
			assert.EqualError(t, err, "circular dependency detected: di.cycleA -> di.cycleB -> di.cycleA")
		})
	})

	t.Run("transients return error instead of overflowing the stack", func(t *testing.T) {
		container := New()
		bindAB(container, WithTransient())

		withTimeout(t, func() {
			var b cycleB
			err := container.Resolve(&b)
			assert.EqualError(t, err, "circular dependency detected: di.cycleB -> di.cycleA -> di.cycleB")
		})
	})

	t.Run("longer cycle reports only the cycle", func(t *testing.T) {
		container := New()
		require.NoError(t, container.Bind(func(cycleB) cycleA { return cycleImpl{} }))
		require.NoError(t, container.Bind(func(cycleC) cycleB { return cycleImpl{} }))
		require.NoError(t, container.Bind(func(cycleB) cycleC { return cycleImpl{} }))

		withTimeout(t, func() {
			var a cycleA
			err := container.Resolve(&a)
			assert.EqualError(t, err, "circular dependency detected: di.cycleB -> di.cycleC -> di.cycleB")
		})
	})

	t.Run("concurrent resolves from both ends of a cycle", func(t *testing.T) {
		container := New()
		bindAB(container)

		withTimeout(t, func() {
			var wg sync.WaitGroup
			for i := 0; i < 50; i++ {
				wg.Add(2)
				go func() {
					defer wg.Done()
					var a cycleA
					assert.Error(t, container.Resolve(&a))
				}()
				go func() {
					defer wg.Done()
					var b cycleB
					assert.Error(t, container.Resolve(&b))
				}()
			}
			wg.Wait()
		})
	})

	t.Run("named root and ResolveAll detect cycles", func(t *testing.T) {
		container := New()
		require.NoError(t, container.Bind(func(cycleB) cycleA { return cycleImpl{} }, WithName("named")))
		require.NoError(t, container.Bind(func(cycleA) cycleB { return cycleImpl{} }))
		require.NoError(t, container.Bind(func(cycleB) cycleA { return cycleImpl{} }))

		withTimeout(t, func() {
			var a cycleA
			err := container.ResolveNamed(&a, "named")
			assert.EqualError(t, err, "circular dependency detected: di.cycleB -> di.cycleA -> di.cycleB")

			var all []cycleA
			assert.Error(t, container.ResolveAll(&all))
		})
	})

	t.Run("eager binding into a cycle fails the bind", func(t *testing.T) {
		container := New()
		require.NoError(t, container.Bind(func(cycleC) cycleB { return cycleImpl{} }))
		require.NoError(t, container.Bind(func(cycleB) cycleC { return cycleImpl{} }))

		withTimeout(t, func() {
			err := container.Bind(func(cycleB) cycleA { return cycleImpl{} }, WithEager())
			assert.EqualError(t, err, "circular dependency detected: di.cycleB -> di.cycleC -> di.cycleB")
		})
	})

	t.Run("rebinding that creates a cycle is detected after a successful check", func(t *testing.T) {
		container := New()
		require.NoError(t, container.Bind(func(cycleB) cycleA { return cycleImpl{} }, WithTransient()))
		require.NoError(t, container.Bind(func() cycleB { return cycleImpl{} }, WithTransient()))

		var a cycleA
		require.NoError(t, container.Resolve(&a))

		require.NoError(t, container.Bind(func(cycleA) cycleB { return cycleImpl{} }, WithTransient()))
		withTimeout(t, func() {
			assert.Error(t, container.Resolve(&a))
		})
	})

	t.Run("Lazy breaks the cycle", func(t *testing.T) {
		container := New()
		require.NoError(t, container.Bind(func(Lazy[cycleB]) cycleA { return cycleImpl{} }))
		require.NoError(t, container.Bind(func(cycleA) cycleB { return cycleImpl{} }))

		withTimeout(t, func() {
			var b cycleB
			assert.NoError(t, container.Resolve(&b))
		})
	})

	t.Run("shared dependency is not a cycle", func(t *testing.T) {
		container := New()
		require.NoError(t, container.Bind(func() cycleC { return cycleImpl{} }))
		require.NoError(t, container.Bind(func(cycleC) cycleB { return cycleImpl{} }))
		require.NoError(t, container.Bind(func(cycleB, cycleC) cycleA { return cycleImpl{} }))

		var a cycleA
		assert.NoError(t, container.Resolve(&a))
	})
}
