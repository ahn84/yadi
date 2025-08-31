# YADI - Yet Another Dependency Injection

A modern, type-safe dependency injection library for Go that leverages generics and reflection to provide clean and efficient dependency injection.

## Features

- ✅ **Singleton-First Design**: Singleton instances by default for better performance and resource management.
- ✅ **Clean API**: Simple, idiomatic Go interface without verbose generics.
- ✅ **Automatic Dependency Resolution**: No manual wiring required.
- ✅ **Thread-Safe**: Concurrent binding and resolution.
- ✅ **Type Inference**: Automatic type detection from function signatures.
- ✅ **Lazy Resolution**: Built-in `Lazy[T]` type to handle circular dependencies.
- ✅ **Resolve All**: Resolve all instances of an interface.
- ✅ **Comprehensive Error Handling**: Clear, descriptive error messages.
- ✅ **Minimal Overhead**: Efficient reflection-based resolution.

## Installation

```bash
go get github.com/ahn84/yadi
```

## Quick Start

```go
package main

import (
	"fmt"
	"github.com/ahn84/yadi"
)

// Define your interfaces
type Database interface {
	Query(sql string) string
}

type UserService interface {
	GetUser(id int) string
}

// Implement your types
type postgresDB struct{}

func (p *postgresDB) Query(sql string) string {
	return "user data"
}

type userService struct {
	db Database
}

func (u *userService) GetUser(id int) string {
	return u.db.Query("SELECT * FROM users")
}

func main() {
	// Bind dependencies using the global container
	yadi.Bind(func() Database {
		return &postgresDB{}
	})

	yadi.Bind(func(db Database) UserService {
		return &userService{db: db}
	})

	// Resolve using pointer
	var userSvc UserService
	err := yadi.Resolve(&userSvc)
	if err != nil {
		panic(err)
	}

	// Use your service
	user := userSvc.GetUser(1)
	fmt.Println(user)
}
```

For more detailed examples, see the [examples](./examples) directory.

## API Reference

### Core Methods

#### `Bind(resolver interface{}, options ...BindOption) error`

Registers a factory function. The return type is automatically detected from the function signature. Creates singleton instances by default.

**Available Options:**
- `WithSingleton()`: Creates a singleton (default).
- `WithTransient()`: Creates transient instances (new instance every time).
- `WithName(string)`: Names the binding for multiple implementations.
- `WithEager()`: Creates instance immediately during binding.

#### `Resolve(target interface{}) error`

Resolves a dependency into the provided pointer.

#### `ResolveNamed(target interface{}, name string) error`

Resolves a named dependency into the provided pointer.

#### `ResolveAll(target interface{}) error`

Resolves all instances of a given type into the provided slice pointer.

### `Lazy[T]` for Circular Dependencies

YADI provides a `Lazy[T]` type to handle circular dependencies gracefully.

```go
type ServiceA struct {
    ServiceB di.Lazy[ServiceB]
}

type ServiceB struct {
    ServiceA *ServiceA
}

// Bind services with circular dependency
di.Bind(func(serviceB di.Lazy[ServiceB]) *ServiceA {
    return &ServiceA{ServiceB: serviceB}
})
di.Bind(func(serviceA *ServiceA) *ServiceB {
    return &ServiceB{ServiceA: serviceA}
})

// Resolve the services
var serviceA *ServiceA
di.Resolve(&serviceA)

// Resolve the lazy dependency
serviceB, err := serviceA.ServiceB.Resolve()
```

### Convenience Methods

- `BindTransient(resolver interface{}, options ...BindOption) error`
- `BindNamed(name string, resolver interface{}, options ...BindOption) error`
- `BindNamedTransient(name string, resolver interface{}, options ...BindOption) error`

### Container Methods

- `New() *Container`: Creates a new dependency injection container.
- `Clear()`: Removes all bindings from the container.

## Examples

- [Simple Usage](./examples/simple)
- [Named Bindings](./examples/named)
- [Transient vs Singleton](./examples/transient)
- [Eager Initialization](./examples/eager)
- [Resolve All](./examples/resolveall)
- [Lazy Resolution (Circular Dependencies)](./examples/lazy)

## Design Principles

1.  **Singleton-First**: Singleton instances by default for better resource management and performance.
2.  **Clean API**: Simple, idiomatic Go without verbose generic syntax.
3.  **Type Inference**: Automatic type detection from function signatures.
4.  **Performance Conscious**: Minimal runtime overhead with optimized reflection.
5.  **Developer Experience**: Simple, intuitive API with clear error messages.
6.  **Flexibility**: Support for complex dependency graphs and error handling.

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](LICENSE) file for details.