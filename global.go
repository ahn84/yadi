package di

var global = New()

// Bind registers a factory function in the global container.
// The resolver function's parameters will be automatically resolved when the return type is requested.
func Bind(resolver interface{}, options ...BindOption) error {
	return global.Bind(resolver, options...)
}

// Resolve returns an instance from the global container by setting the value of the provided pointer.
// The target must be a pointer to the type you want to resolve.
func Resolve(target interface{}) error {
	return global.Resolve(target)
}

// ResolveNamed returns a named instance from the global container by setting the value of the provided pointer.
// The target must be a pointer to the type you want to resolve.
func ResolveNamed(target interface{}, name string) error {
	return global.ResolveNamed(target, name)
}

// BindTransient is a convenience method for binding a transient instance in the global container.
func BindTransient(resolver interface{}, options ...BindOption) error {
	return global.BindTransient(resolver, options...)
}

// BindNamed is a convenience method for binding with a name in the global container.
func BindNamed(name string, resolver interface{}, options ...BindOption) error {
	return global.BindNamed(name, resolver, options...)
}

// BindNamedTransient is a convenience method for binding a named transient instance in the global container.
func BindNamedTransient(name string, resolver interface{}, options ...BindOption) error {
	return global.BindNamedTransient(name, resolver, options...)
}

// Clear removes all bindings from the global container.
func Clear() {
	global.Clear()
}
