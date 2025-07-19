# YADI - Yet Another Dependency Injection

A modern, type-safe dependency injection library for Go that leverages generics and reflection to provide clean and efficient dependency injection.

## Features

- ✅ **Clean API**: Simple, idiomatic Go interface without verbose generics
- ✅ **Automatic dependency resolution**: No manual wiring required
- ✅ **Thread-safe**: Concurrent binding and resolution
- ✅ **Type inference**: Automatic type detection from function signatures
- ✅ **Comprehensive error handling**: Clear, descriptive error messages
- ✅ **Minimal overhead**: Efficient reflection-based resolution

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
    // Create container
    container := di.New()
    
    // Bind dependencies - types inferred from function signatures
    container.Bind(func() Database {
        return &postgresDB{}
    })
    
    container.Bind(func(db Database) UserService {
        return &userService{db: db}
    })
    
    // Resolve using pointer
    var userSvc UserService
    err := container.Resolve(&userSvc)
    if err != nil {
        panic(err)
    }
    
    // Use your service
    user := userSvc.GetUser(1)
    fmt.Println(user)
}
```

## API Reference

### Core Methods

#### `Bind(resolver interface{}) error`

Registers a factory function. The return type is automatically detected from the function signature.

```go
// Simple binding
err := container.Bind(func() Database {
    return &postgresDB{}
})

// Binding with dependencies (automatically resolved)
err := container.Bind(func(db Database, logger Logger) UserService {
    return &userService{db: db, logger: logger}
})

// Binding with error handling
err := container.Bind(func() (Database, error) {
    db := &postgresDB{}
    if err := db.Connect(); err != nil {
        return nil, err
    }
    return db, nil
})
```

#### `Resolve(target interface{}) error`

Resolves a dependency into the provided pointer, automatically resolving all dependencies.

```go
var userService UserService
err := container.Resolve(&userService)
if err != nil {
    // Handle resolution error
}
```

### Container Methods

#### `New() *Container`

Creates a new dependency injection container.

```go
container := di.New()
```

#### `Clear()`

Removes all bindings from the container.

```go
container.Clear()
```

## Advanced Usage

### Complex Dependency Chains

YADI automatically resolves complex dependency chains:

```go
// Service with multiple dependencies
container.Bind(func(
    userSvc UserService, 
    paymentSvc PaymentService,
    logger Logger,
) OrderService {
    return &orderService{
        userSvc:    userSvc,
        paymentSvc: paymentSvc,
        logger:     logger,
    }
})

// All dependencies are resolved automatically
var orderSvc OrderService
err := container.Resolve(&orderSvc)
```

### Error Handling

```go
// Binding that can fail
container.Bind(func() (Database, error) {
    db := &postgresDB{}
    if err := db.Connect(); err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }
    return db, nil
})

// Resolution with error handling
var db Database
err := container.Resolve(&db)
if err != nil {
    log.Fatalf("Database resolution failed: %v", err)
}
```

### Thread Safety

YADI is fully thread-safe for concurrent binding and resolution:

```go
// Safe to call from multiple goroutines
go func() {
    var userSvc UserService
    container.Resolve(&userSvc)
    // Use userSvc...
}()

go func() {
    var orderSvc OrderService
    container.Resolve(&orderSvc)
    // Use orderSvc...
}()
```

## Design Principles

1. **Clean API**: Simple, idiomatic Go without verbose generic syntax
2. **Type Inference**: Automatic type detection from function signatures
3. **Performance Conscious**: Minimal runtime overhead with optimized reflection
4. **Developer Experience**: Simple, intuitive API with clear error messages
5. **Flexibility**: Support for complex dependency graphs and error handling

## Error Types

- **Missing Dependency**: When a required dependency is not bound
- **Circular Dependency**: When services depend on each other in a cycle
- **Invalid Resolver**: When the resolver function is malformed
- **Resolution Error**: When a resolver function returns an error

## Performance

- **Binding**: O(1) average case registration
- **Resolution**: O(D) where D is dependency depth
- **Memory**: ~100 bytes overhead per binding
- **Thread Safety**: Read-optimized with minimal lock contention

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Status**: Alpha - API may change
**Go Version**: Requires Go 1.18+ (for generics support)
