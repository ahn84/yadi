package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test interfaces and structs
type Database interface {
	Connect() error
}

type UserService interface {
	GetUser(id int) string
}

type Logger interface {
	Log(message string)
}

type mockDatabase struct {
	connected bool
}

func (m *mockDatabase) Connect() error {
	m.connected = true
	return nil
}

type userServiceImpl struct {
	db Database
}

func (u *userServiceImpl) GetUser(id int) string {
	return "user"
}

type loggerImpl struct {
	messages []string
}

func (l *loggerImpl) Log(message string) {
	l.messages = append(l.messages, message)
}

// Complex service with multiple dependencies
type OrderService interface {
	CreateOrder(userID int) string
}

type orderServiceImpl struct {
	userService UserService
	db          Database
	logger      Logger
}

func (o *orderServiceImpl) CreateOrder(userID int) string {
	o.logger.Log("Creating order")
	user := o.userService.GetUser(userID)
	return "order for " + user
}

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
	t.Run("transient instances are different", func(t *testing.T) {
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

		// Should be different instances
		assert.NotSame(t, db1, db2)
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
