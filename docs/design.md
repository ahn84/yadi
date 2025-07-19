# YADI - Design Document

## Overview

YADI (Yet Another Dependency Injection) is a modern dependency injection container for Go that combines the power of reflection with Go's type system to provide a clean, type-safe, and efficient dependency injection solution.

## Design Philosophy

### 1. Type Safety First
- **Generic API**: Leverage Go 1.18+ generics to provide compile-time type safety
- **No Type Assertions**: Eliminate runtime type assertions in user code
- **Clear Contracts**: Function signatures that clearly express dependencies and returns

### 2. Reflection for Flexibility
- **Runtime Discovery**: Use reflection to analyze function signatures and resolve dependencies
- **Automatic Wiring**: No manual configuration for simple dependency chains
- **Dynamic Resolution**: Support for complex dependency graphs

### 3. Performance Conscious
- **Lazy Resolution**: Only resolve dependencies when needed
- **Singleton Caching**: Cache singleton instances to avoid repeated construction
- **Minimal Allocations**: Optimize for low memory overhead during resolution

### 4. Developer Experience
- **Simple API**: Minimal learning curve with intuitive method names
- **Clear Errors**: Descriptive error messages with dependency chain information
- **Debugging Support**: Tools to understand and visualize dependency graphs

## Core Architecture

### Container Structure

```go
type Container struct {
    bindings map[reflect.Type]map[string]*binding
    lock     sync.RWMutex
}
```

**Design Rationale:**
- **Two-level mapping**: `Type -> Name -> Binding` allows both default and named bindings
- **Thread-safe**: RWMutex enables concurrent reads while protecting writes
- **Type-based indexing**: Uses `reflect.Type` as primary key for O(1) lookup

### Binding Structure

```go
type binding struct {
    resolver  any    // factory function or value
    concrete  any    // cached instance for singletons
    singleton bool   // lifecycle management flag
    scope     string // future: scoping support
}
```

**Design Rationale:**
- **Flexible Resolver**: Can hold factory functions or direct values
- **Lazy Evaluation**: `concrete` is only populated when needed
- **Lifecycle Awareness**: `singleton` flag controls caching behavior
- **Extensible**: `scope` field prepared for future scoping features

## Dependency Resolution Algorithm

### 1. Function Analysis
```go
func (c *Container) resolveArguments(function interface{}) ([]reflect.Value, error)
```

**Process:**
1. Extract function type using reflection
2. Iterate through each input parameter
3. Look up binding for parameter type
4. Recursively resolve dependencies
5. Return prepared argument list

**Benefits:**
- **Automatic Discovery**: No manual wiring configuration
- **Deep Resolution**: Handles nested dependencies automatically
- **Type Matching**: Uses exact type matching for precision

### 2. Circular Dependency Prevention
The current design implicitly prevents circular dependencies through the call stack, but future enhancements will add explicit detection.

### 3. Error Propagation
Errors bubble up through the dependency chain, providing clear context about which dependency failed to resolve.

## API Design Principles

### Generic Methods
```go
func Bind[T any](resolver func(...) T) error
func Resolve[T]() (T, error)
```

**Benefits:**
- **Compile-time Safety**: Type mismatches caught at compile time
- **IntelliSense Support**: IDEs can provide accurate type hints
- **No Casting**: Eliminates need for type assertions in user code

### Error Handling
All operations return explicit errors following Go conventions:
- **Explicit Errors**: No panic-based APIs for better control
- **Contextual Information**: Errors include dependency chain context
- **Recoverable Failures**: Applications can handle DI failures gracefully

## Binding Strategies

### 1. Factory Functions
```go
container.Bind(func(db Database) UserService { 
    return &userServiceImpl{db: db} 
})
```

**Advantages:**
- **Lazy Construction**: Objects created only when needed
- **Dependency Injection**: Parameters automatically resolved
- **Flexible Logic**: Custom construction logic possible

### 2. Direct Values
```go
container.BindValue(&Config{...})
```

**Use Cases:**
- **Configuration Objects**: Pre-constructed configuration
- **External Resources**: Third-party objects
- **Testing**: Mock objects and test doubles

### 3. Named Bindings
```go
container.BindNamed("redis", func() Cache { return &RedisCache{} })
container.BindNamed("memory", func() Cache { return &MemoryCache{} })
```

**Benefits:**
- **Multiple Implementations**: Same interface, different implementations
- **Environment-specific**: Different implementations for dev/prod
- **Feature Flags**: Conditional implementation selection

## Lifecycle Management

### Singleton Pattern
```go
func (b *binding) resolve(c *Container) (any, error) {
    if b.concrete != nil {
        return b.concrete, nil  // Return cached instance
    }
    val, err := c.callResolver(b.resolver)
    if b.singleton {
        b.concrete = val  // Cache for future use
    }
    return val, err
}
```

**Design Decisions:**
- **Lazy Initialization**: Singletons created on first access
- **Thread-safe Caching**: Protected by container's mutex
- **Memory Efficient**: Non-singletons don't use cache storage

### Future: Scoped Instances
- **Request Scope**: One instance per HTTP request
- **Session Scope**: One instance per user session
- **Custom Scopes**: User-defined lifecycle management

## Thread Safety Model

### Read-Heavy Optimization
- **RWMutex**: Multiple concurrent reads, exclusive writes
- **Resolution Phase**: Only read locks during dependency resolution
- **Registration Phase**: Write locks only during binding registration

### Lock Granularity
- **Container-level Locking**: Simpler implementation, prevents deadlocks
- **Future Optimization**: Per-type locking for better concurrency

## Error Design

### Custom Error Types (Planned)
```go
type ErrNotFound struct {
    Type reflect.Type
    Name string
}

type ErrCircularDependency struct {
    Chain []reflect.Type
}
```

**Benefits:**
- **Programmatic Handling**: Applications can respond to specific error types
- **Rich Context**: Detailed information for debugging
- **Error Recovery**: Possible to implement fallback strategies

## Performance Characteristics

### Time Complexity
- **Binding Registration**: O(1) average case
- **Dependency Resolution**: O(D) where D is dependency depth
- **Type Lookup**: O(1) hash map access

### Memory Usage
- **Per Binding**: ~100 bytes overhead
- **Singleton Cache**: One pointer per singleton instance
- **Reflection Cache**: Go runtime manages type reflection caching

### Optimization Opportunities
1. **Pre-computed Resolution Plans**: Cache dependency resolution paths
2. **Code Generation**: Generate direct instantiation code
3. **Pool Reuse**: Reuse reflection value slices

## Comparison with Other Approaches

### vs Manual Dependency Injection
**Advantages:**
- **Reduced Boilerplate**: No manual wiring code
- **Automatic Discovery**: Dependencies resolved automatically
- **Centralized Configuration**: All bindings in one place

**Trade-offs:**
- **Runtime Overhead**: Reflection has performance cost
- **Learning Curve**: DI concepts need to be understood

### vs Wire (Compile-time DI)
**Advantages:**
- **Runtime Flexibility**: Change bindings without recompilation
- **Dynamic Configuration**: Conditional binding based on runtime state
- **Simpler Setup**: No code generation step

**Trade-offs:**
- **Performance**: Runtime resolution vs compile-time optimization
- **Error Detection**: Some errors only caught at runtime

## Future Design Considerations

### Code Generation Path
Generate optimized resolution code for production builds:
```go
// Generated code
func ResolveUserService() *UserService {
    db := ResolveDatabase()
    return &userServiceImpl{db: db}
}
```

### Plugin Architecture
Support for extension points:
- **Custom Resolvers**: User-defined resolution strategies
- **Lifecycle Hooks**: Pre/post construction callbacks
- **Interceptors**: Method call interception

### Configuration-driven DI
Support for external configuration:
```yaml
bindings:
  - type: UserService
    implementation: userServiceImpl
    scope: singleton
    dependencies:
      - Database
```

## Testing Strategy

### Unit Testing
- **Mock Container**: Simplified container for testing
- **Dependency Overrides**: Replace dependencies for testing
- **Isolation**: Each test gets clean container state

### Integration Testing
- **Real Dependencies**: Test with actual implementations
- **Startup Validation**: Ensure all dependencies can be resolved
- **Performance Testing**: Measure resolution time and memory usage

## Security Considerations

### Input Validation
- **Function Signature Validation**: Ensure resolver functions are well-formed
- **Circular Dependency Detection**: Prevent infinite recursion
- **Type Safety**: Prevent type confusion attacks

### Resource Management
- **Memory Leaks**: Proper cleanup of singleton instances
- **Goroutine Safety**: Prevent race conditions
- **Resource Limits**: Prevent unbounded dependency chains

---

**Last Updated:** July 20, 2025  
**Author:** YADI Development Team  
**Version:** 1.0
