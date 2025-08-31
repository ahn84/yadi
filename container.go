package di

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// BindOption represents a configuration option for binding
type BindOption func(*bindConfig)

// bindConfig holds the configuration for a binding
type bindConfig struct {
	name      string
	singleton bool
	lazy      bool
}

// WithName sets a name for the binding, allowing multiple implementations of the same interface
func WithName(name string) BindOption {
	return func(config *bindConfig) {
		config.name = name
	}
}

// WithSingleton makes the binding a singleton (same instance returned on every resolve) - this is now the default
func WithSingleton() BindOption {
	return func(config *bindConfig) {
		config.singleton = true
	}
}

// WithTransient makes the binding transient (new instance on every resolve) - explicit override of singleton default
func WithTransient() BindOption {
	return func(config *bindConfig) {
		config.singleton = false
	}
}

// WithLazy makes the binding lazy (instance created only when first requested) - this is the default
func WithLazy() BindOption {
	return func(config *bindConfig) {
		config.lazy = true
	}
}

// WithEager makes the binding eager (instance created immediately during binding)
func WithEager() BindOption {
	return func(config *bindConfig) {
		config.lazy = false
	}
}

type binding struct {
	resolver  any        // factory function or value
	concrete  any        // concrete type
	singleton bool       // whether the binding is a singleton
	mutex     sync.Mutex // protects concrete for singleton instances
}

func (b *binding) resolve(c *Container) (any, error) {
	// For singleton bindings, use mutex for thread safety
	if b.singleton {
		b.mutex.Lock()
		defer b.mutex.Unlock()

		// Check if we already have a cached instance
		if b.concrete != nil {
			return b.concrete, nil
		}

		// Create the instance
		val, err := c.callResolver(b.resolver)
		if err != nil {
			return nil, err
		}

		// Cache it for future use
		b.concrete = val
		return val, nil
	}

	// For transient bindings, just create a new instance each time
	return c.callResolver(b.resolver)
}

type Container struct {
	bindings map[reflect.Type]map[string]*binding
	lock     sync.RWMutex
}

func New() *Container {
	return &Container{
		bindings: make(map[reflect.Type]map[string]*binding),
	}
}

func (c *Container) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.bindings = make(map[reflect.Type]map[string]*binding)
}

// Bind registers a factory function in the container.
// The resolver function's parameters will be automatically resolved when the return type is requested.
func (c *Container) Bind(resolver interface{}, options ...BindOption) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Apply default configuration
	config := &bindConfig{
		name:      "",
		singleton: true,
		lazy:      true,
	}

	// Apply provided options
	for _, option := range options {
		option(config)
	}

	return c.bind(resolver, config.name, config.singleton, config.lazy)
}

// Resolve returns an instance by setting the value of the provided pointer.
// The target must be a pointer to the type you want to resolve.
func (c *Container) Resolve(target interface{}) error {
	return c.ResolveNamed(target, "")
}

// ResolveNamed returns a named instance by setting the value of the provided pointer.
// The target must be a pointer to the type you want to resolve.
func (c *Container) ResolveNamed(target interface{}, name string) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	targetType := targetValue.Elem().Type()
	if bindings, exists := c.bindings[targetType]; exists {
		if binding, exists := bindings[name]; exists {
			instance, err := binding.resolve(c)
			if err != nil {
				return err
			}
			targetValue.Elem().Set(reflect.ValueOf(instance))
			return nil
		}
	}
	return fmt.Errorf("no binding found for type %s with name '%s'", targetType.String(), name)
}

// BindTransient is a convenience method for binding a transient instance
func (c *Container) BindTransient(resolver interface{}, options ...BindOption) error {
	allOptions := append([]BindOption{WithTransient()}, options...)
	return c.Bind(resolver, allOptions...)
}

// BindNamed is a convenience method for binding with a name
func (c *Container) BindNamed(name string, resolver interface{}, options ...BindOption) error {
	allOptions := append([]BindOption{WithName(name)}, options...)
	return c.Bind(resolver, allOptions...)
}

// BindNamedTransient is a convenience method for binding a named transient instance
func (c *Container) BindNamedTransient(name string, resolver interface{}, options ...BindOption) error {
	allOptions := append([]BindOption{WithName(name), WithTransient()}, options...)
	return c.Bind(resolver, allOptions...)
}

// calls the resolver function
func (c *Container) callResolver(function interface{}) (interface{}, error) {
	arguments, err := c.resolveArguments(function)
	if err != nil {
		return nil, err
	}

	values := reflect.ValueOf(function).Call(arguments)
	if len(values) == 2 && values[1].CanInterface() {
		if err, ok := values[1].Interface().(error); ok {
			return values[0].Interface(), err
		}
	}
	return values[0].Interface(), nil
}

// arguments returns the list of resolved arguments for a function.
func (c *Container) resolveArguments(function interface{}) ([]reflect.Value, error) {
	refFunc := reflect.TypeOf(function)
	argNum := refFunc.NumIn()
	arguments := make([]reflect.Value, argNum)

	for i := 0; i < argNum; i++ {
		argType := refFunc.In(i)
		if bound, exist := c.bindings[argType][""]; exist {
			instance, err := bound.resolve(c)
			if err != nil {
				return nil, err
			}
			arguments[i] = reflect.ValueOf(instance)
		} else {
			return nil, errors.New("failed resolving argument " + argType.String())
		}
	}

	return arguments, nil
}

// bind maps an abstraction to concrete and instantiates if it is a singleton binding.
func (c *Container) bind(resolver interface{}, name string, isSingleton bool, isLazy bool) error {
	reflectedResolver := reflect.TypeOf(resolver)
	if reflectedResolver.Kind() != reflect.Func {
		return errors.New("container: the resolver must be a function")
	}

	if reflectedResolver.NumOut() > 0 {
		if _, exist := c.bindings[reflectedResolver.Out(0)]; !exist {
			c.bindings[reflectedResolver.Out(0)] = make(map[string]*binding)
		}
	}

	if err := c.validateResolverFunction(reflectedResolver); err != nil {
		return err
	}

	var concrete interface{}
	if !isLazy {
		var err error
		concrete, err = c.callResolver(resolver)
		if err != nil {
			return err
		}
	}

	if isSingleton {
		c.bindings[reflectedResolver.Out(0)][name] = &binding{resolver: resolver, concrete: concrete, singleton: isSingleton}
	} else {
		c.bindings[reflectedResolver.Out(0)][name] = &binding{resolver: resolver, singleton: isSingleton}
	}

	return nil
}

func (c *Container) validateResolverFunction(funcType reflect.Type) error {
	retCount := funcType.NumOut()

	if retCount == 0 || retCount > 2 {
		return errors.New("need exactly one or two return values")
	}

	resolveType := funcType.Out(0)
	for i := 0; i < funcType.NumIn(); i++ {
		if funcType.In(i) == resolveType {
			return fmt.Errorf("can't depend on return type")
		}
	}

	return nil
}
