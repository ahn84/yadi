package di

import (
	"reflect"
	"strings"
)

// Lazy is a helper type for lazy dependency resolution.
type Lazy[T any] struct {
	Container *Container
}

// Resolve resolves the dependency.
func (l *Lazy[T]) Resolve() (T, error) {
	var instance T
	err := l.Container.Resolve(&instance)
	return instance, err
}

func isLazy(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && strings.HasPrefix(t.Name(), "Lazy[")
}