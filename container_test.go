package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)



func TestContainer_New(t *testing.T) {
	container := New()
	assert.NotNil(t, container)
	assert.NotNil(t, container.bindings)
}

func TestContainer_Bind(t *testing.T) {
	t.Run("bind simple factory function", func(t *testing.T) {
		container := New()

		err := container.Bind(func() Database {
			return &mockDatabase{}
		})

		assert.NoError(t, err)
	})

	t.Run("bind function with dependencies", func(t *testing.T) {
		container := New()

		// Bind dependency first
		err := container.Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		// Bind service that depends on Database
		err = container.Bind(func(db Database) UserService {
			return &userServiceImpl{db: db}
		})

		assert.NoError(t, err)
	})

	t.Run("bind multiple dependencies", func(t *testing.T) {
		container := New()

		// Bind all dependencies
		err := container.Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		err = container.Bind(func(db Database) UserService {
			return &userServiceImpl{db: db}
		})
		require.NoError(t, err)

		err = container.Bind(func() Logger {
			return &loggerImpl{}
		})
		require.NoError(t, err)

		// Bind service with multiple dependencies
		err = container.Bind(func(userService UserService, db Database, logger Logger) OrderService {
			return &orderServiceImpl{
				userService: userService,
				db:          db,
				logger:      logger,
			}
		})

		assert.NoError(t, err)
	})

	t.Run("error when resolver is not a function", func(t *testing.T) {
		container := New()

		err := container.Bind("not a function")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resolver must be a function")
	})

	t.Run("error when function has no return values", func(t *testing.T) {
		container := New()

		err := container.Bind(func() {
			// no return value
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "need exactly one or two return values")
	})

	t.Run("error when function has too many return values", func(t *testing.T) {
		container := New()

		err := container.Bind(func() (Database, Logger, error) {
			return nil, nil, nil
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "need exactly one or two return values")
	})

	t.Run("allow function with error return", func(t *testing.T) {
		container := New()

		err := container.Bind(func() (Database, error) {
			return &mockDatabase{}, nil
		})

		assert.NoError(t, err)
	})

	t.Run("error when function depends on its return type", func(t *testing.T) {
		container := New()

		err := container.Bind(func(db Database) Database {
			return db
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "can't depend on return type")
	})
}

func TestContainer_BindWithOptions(t *testing.T) {
	t.Run("bind with explicit singleton option", func(t *testing.T) {
		container := New()

		err := container.Bind(func() Database {
			return &mockDatabase{}
		}, WithSingleton()) // Explicit singleton (redundant but allowed)

		require.NoError(t, err)

		var db1, db2 Database
		err = container.Resolve(&db1)
		require.NoError(t, err)

		err = container.Resolve(&db2)
		require.NoError(t, err)

		// Should be the same instance for singletons
		assert.Same(t, db1, db2)
	})

	t.Run("bind with name option", func(t *testing.T) {
		container := New()

		err := container.Bind(func() Database {
			return &mockDatabase{connected: false}
		}, WithName("primary"))
		require.NoError(t, err)

		err = container.Bind(func() Database {
			db := &mockDatabase{connected: true}
			return db
		}, WithName("secondary"))
		require.NoError(t, err)

		var primaryDB Database
		err = container.ResolveNamed(&primaryDB, "primary")
		require.NoError(t, err)

		var secondaryDB Database
		err = container.ResolveNamed(&secondaryDB, "secondary")
		require.NoError(t, err)

		// Should be different instances with different states
		assert.False(t, primaryDB.(*mockDatabase).connected)
		assert.True(t, secondaryDB.(*mockDatabase).connected)
	})

	t.Run("bind with multiple options", func(t *testing.T) {
		container := New()

		err := container.Bind(func() Database {
			return &mockDatabase{}
		}, WithName("cached")) // Singleton by default + named

		require.NoError(t, err)

		var db1, db2 Database
		err = container.ResolveNamed(&db1, "cached")
		require.NoError(t, err)

		err = container.ResolveNamed(&db2, "cached")
		require.NoError(t, err)

		// Should be the same instance (singleton + named)
		assert.Same(t, db1, db2)
	})

	t.Run("bind with eager option", func(t *testing.T) {
		container := New()

		called := false
		err := container.Bind(func() Database {
			called = true
			return &mockDatabase{}
		}, WithEager())

		require.NoError(t, err)
		// Should be called immediately due to eager binding
		assert.True(t, called)
	})

	t.Run("bind with lazy option (default)", func(t *testing.T) {
		container := New()

		called := false
		err := container.Bind(func() Database {
			called = true
			return &mockDatabase{}
		}, WithLazy())

		require.NoError(t, err)
		// Should not be called yet due to lazy binding
		assert.False(t, called)

		var db Database
		err = container.Resolve(&db)
		require.NoError(t, err)
		// Now it should be called
		assert.True(t, called)
	})
}

func TestContainer_ConvenienceMethods(t *testing.T) {
	t.Run("BindTransient", func(t *testing.T) {
		container := New()

		err := container.BindTransient(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		var db1, db2 Database
		err = container.Resolve(&db1)
		require.NoError(t, err)

		err = container.Resolve(&db2)
		require.NoError(t, err)

		assert.NotSame(t, db1, db2)
	})

	t.Run("BindNamed", func(t *testing.T) {
		container := New()

		err := container.BindNamed("test", func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		var db Database
		err = container.ResolveNamed(&db, "test")
		require.NoError(t, err)
		assert.NotNil(t, db)
	})

	t.Run("BindNamedTransient", func(t *testing.T) {
		container := New()

		err := container.BindNamedTransient("transient-test", func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		var db1, db2 Database
		err = container.ResolveNamed(&db1, "transient-test")
		require.NoError(t, err)

		err = container.ResolveNamed(&db2, "transient-test")
		require.NoError(t, err)

		assert.NotSame(t, db1, db2)
	})

	t.Run("convenience methods with additional options", func(t *testing.T) {
		container := New()

		called := false
		err := container.BindTransient(func() Database {
			called = true
			return &mockDatabase{}
		}, WithEager())
		require.NoError(t, err)

		// Should be called immediately (eager) and be transient
		assert.True(t, called)

		var db1, db2 Database
		err = container.Resolve(&db1)
		require.NoError(t, err)

		err = container.Resolve(&db2)
		require.NoError(t, err)

		assert.NotSame(t, db1, db2)
	})
}

func TestContainer_NamedResolution(t *testing.T) {
	t.Run("resolve named binding", func(t *testing.T) {
		container := New()

		// Bind multiple implementations of the same interface
		err := container.BindNamed("redis", func() Logger {
			return &loggerImpl{messages: []string{"redis"}}
		})
		require.NoError(t, err)

		err = container.BindNamed("file", func() Logger {
			return &loggerImpl{messages: []string{"file"}}
		})
		require.NoError(t, err)

		var redisLogger Logger
		err = container.ResolveNamed(&redisLogger, "redis")
		require.NoError(t, err)

		var fileLogger Logger
		err = container.ResolveNamed(&fileLogger, "file")
		require.NoError(t, err)

		assert.Equal(t, []string{"redis"}, redisLogger.(*loggerImpl).messages)
		assert.Equal(t, []string{"file"}, fileLogger.(*loggerImpl).messages)
	})

	t.Run("error when named binding not found", func(t *testing.T) {
		container := New()

		var logger Logger
		err := container.ResolveNamed(&logger, "nonexistent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no binding found for type")
		assert.Contains(t, err.Error(), "with name 'nonexistent'")
	})

	t.Run("resolve default binding when name is empty", func(t *testing.T) {
		container := New()

		err := container.Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		var db Database
		err = container.ResolveNamed(&db, "")
		require.NoError(t, err)
		assert.NotNil(t, db)
	})
}

func TestContainer_SingletonBehavior(t *testing.T) {
	t.Run("singleton instances are same by default", func(t *testing.T) {
		container := New()

		err := container.Bind(func() Database {
			return &mockDatabase{}
		}) // Singleton by default
		require.NoError(t, err)

		var db1, db2 Database
		err = container.Resolve(&db1)
		require.NoError(t, err)

		err = container.Resolve(&db2)
		require.NoError(t, err)

		assert.Same(t, db1, db2)
	})

	t.Run("transient instances are different with explicit option", func(t *testing.T) {
		container := New()

		err := container.Bind(func() Database {
			return &mockDatabase{}
		}, WithTransient()) // Explicit transient
		require.NoError(t, err)

		var db1, db2 Database
		err = container.Resolve(&db1)
		require.NoError(t, err)

		err = container.Resolve(&db2)
		require.NoError(t, err)

		assert.NotSame(t, db1, db2)
	})

	t.Run("singleton with dependencies", func(t *testing.T) {
		container := New()

		// Bind dependency as singleton (default)
		err := container.Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		// Bind service that depends on singleton database (also singleton by default)
		err = container.Bind(func(db Database) UserService {
			return &userServiceImpl{db: db}
		})
		require.NoError(t, err)

		var svc1, svc2 UserService
		err = container.Resolve(&svc1)
		require.NoError(t, err)

		err = container.Resolve(&svc2)
		require.NoError(t, err)

		// Services should be the same (singleton) and should share the same database instance
		assert.Same(t, svc1, svc2)
		assert.Same(t, svc1.(*userServiceImpl).db, svc2.(*userServiceImpl).db)
	})
}

func TestContainer_Resolve(t *testing.T) {
	t.Run("resolve simple binding", func(t *testing.T) {
		container := New()

		err := container.Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		var db Database
		err = container.Resolve(&db)

		assert.NoError(t, err)
		assert.NotNil(t, db)
		assert.IsType(t, &mockDatabase{}, db)
	})

	t.Run("resolve with dependencies", func(t *testing.T) {
		container := New()

		// Bind dependencies
		err := container.Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		err = container.Bind(func(db Database) UserService {
			return &userServiceImpl{db: db}
		})
		require.NoError(t, err)

		// Resolve service
		var userService UserService
		err = container.Resolve(&userService)

		assert.NoError(t, err)
		assert.NotNil(t, userService)

		// Verify the service works
		user := userService.GetUser(1)
		assert.Equal(t, "user", user)
	})

	t.Run("resolve complex dependency chain", func(t *testing.T) {
		container := New()

		// Bind all dependencies
		err := container.Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		err = container.Bind(func(db Database) UserService {
			return &userServiceImpl{db: db}
		})
		require.NoError(t, err)

		err = container.Bind(func() Logger {
			return &loggerImpl{}
		})
		require.NoError(t, err)

		err = container.Bind(func(userService UserService, db Database, logger Logger) OrderService {
			return &orderServiceImpl{
				userService: userService,
				db:          db,
				logger:      logger,
			}
		})
		require.NoError(t, err)

		// Resolve complex service
		var orderService OrderService
		err = container.Resolve(&orderService)

		assert.NoError(t, err)
		assert.NotNil(t, orderService)

		// Verify the service works
		order := orderService.CreateOrder(1)
		assert.Equal(t, "order for user", order)
	})

	t.Run("error when binding not found", func(t *testing.T) {
		container := New()

		var db Database
		err := container.Resolve(&db)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no binding found")
	})

	t.Run("error when dependency not found", func(t *testing.T) {
		container := New()

		// Bind service without its dependency
		err := container.Bind(func(db Database) UserService {
			return &userServiceImpl{db: db}
		})
		require.NoError(t, err)

		var userService UserService
		err = container.Resolve(&userService)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed resolving argument")
	})

	t.Run("handle resolver function errors", func(t *testing.T) {
		container := New()

		// Bind function that returns an error
		err := container.Bind(func() (Database, error) {
			return nil, errors.New("database connection failed")
		})
		require.NoError(t, err)

		var db Database
		err = container.Resolve(&db)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection failed")
	})

	t.Run("error when target is not a pointer", func(t *testing.T) {
		container := New()

		err := container.Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		var db Database
		err = container.Resolve(db) // Pass value instead of pointer

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a pointer")
	})
}

func TestContainer_TransientInstances(t *testing.T) {
	t.Run("singleton instances are same by default", func(t *testing.T) {
		container := New()

		err := container.Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		var db1, db2 Database
		err = container.Resolve(&db1)
		require.NoError(t, err)

		err = container.Resolve(&db2)
		require.NoError(t, err)

		// Should be the same instance by default (singleton)
		assert.Same(t, db1, db2)
	})
}

func TestContainer_Clear(t *testing.T) {
	t.Run("clear removes all bindings", func(t *testing.T) {
		container := New()

		err := container.Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		container.Clear()

		var db Database
		err = container.Resolve(&db)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no binding found")
	})
}

func TestContainer_ThreadSafety(t *testing.T) {
	t.Run("concurrent binding and resolution", func(t *testing.T) {
		container := New()

		// Pre-bind a dependency
		err := container.Bind(func() Database {
			return &mockDatabase{}
		})
		require.NoError(t, err)

		err = container.Bind(func(db Database) UserService {
			return &userServiceImpl{db: db}
		})
		require.NoError(t, err)

		// Test concurrent resolution
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				var userService UserService
				err := container.Resolve(&userService)
				assert.NoError(t, err)
				assert.NotNil(t, userService)
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
