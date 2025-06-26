package di

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

type binding struct {
	resolver  any    // factory function or value
	concrete  any    // concrete type
	singleton bool   // whether the binding is a singleton
	scope     string // binding scope
}

func (b *binding) resolve(c *Container) (any, error) {
	if b.concrete != nil {
		return b.concrete, nil
	}
	val, err := c.callResolver(b.resolver)
	if b.singleton {
		b.concrete = val
	}
	return val, err
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
		abstraction := refFunc.In(i)
		if concrete, exist := c.bindings[abstraction][""]; exist {
			instance, err := concrete.resolve(c)
			if err != nil {
				return nil, err
			}
			arguments[i] = reflect.ValueOf(instance)
		} else {
			return nil, errors.New("failed resolving " + abstraction.String())
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
