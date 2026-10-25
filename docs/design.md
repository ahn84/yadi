# YADI Technical Design

This document describes how YADI (package `di`, module `github.com/ahn84/yadi`) is built: its data model, how binding and resolution work, the concurrency model, and the design of interface binding with `As[T]`.

## Goals

- **Constructor-based wiring.** A binding is a plain Go function. Its parameters are its dependencies and its first return value is what it provides. No struct tags, no code generation.
- **Type inference.** The bound type comes from the function signature, so the common case needs no generics or type names at the call site.
- **Singleton by default.** Most application services are long-lived, so instances are cached unless the binding asks otherwise.
- **Small surface.** One `Container` type, a few `Bind*`/`Resolve*` methods, and functional options for everything else.

## Package layout

| File | Contents |
| --- | --- |
| `container.go` | `Container`, `binding`, bind options, binding and resolution logic |
| `global.go` | Package-level functions that delegate to a shared global container |
| `lazy.go` | `Lazy[T]` deferred dependency type |
| `examples/` | Runnable programs, one per feature |

## Data model

```go
type Container struct {
	bindings map[reflect.Type]map[string]*binding
	lock     sync.RWMutex
}

type binding struct {
	resolver   any           // the constructor function
	concrete   any           // cached instance (singletons only)
	singleton  bool
	mutex      sync.Mutex    // guards concrete
	typ        reflect.Type  // type the binding is registered under
	name       string        // name the binding is registered under
	checkedGen atomic.Uint64 // generation of the last successful cycle check
}
```

`Container` also keeps a `generation uint64` counter that invalidates cycle checks (see [Cycle detection](#cycle-detection)).

Bindings are keyed by **type**, then by **name**. The unnamed binding uses the empty name `""`. Binding the same type and name again replaces the earlier binding.

## Binding

`Container.Bind(resolver, options...)` builds a `bindConfig` from the options, starting from these defaults:

| Field | Default | Options |
| --- | --- | --- |
| `name` | `""` | `WithName` |
| `singleton` | `true` | `WithSingleton`, `WithTransient` |
| `lazy` | `true` | `WithLazy`, `WithEager` |
| `as` | `nil` (use the return type) | `As[T]` |

`BindTransient`, `BindNamed` and `BindNamedTransient` just prepend options and call `Bind`. Options are applied in order, so a later option overrides an earlier one.

`bind` then runs these steps while holding the container's write lock:

1. **Check the shape.** The resolver must be a function with one or two return values. If there are two, the second is treated as an `error`.
2. **Reject self-dependency.** No parameter may have the same type as the first return value.
3. **Work out the key type.** This is the first return value's type, or `T` when `As[T]` is set (see [Interface binding](#interface-binding-ast)).
4. **Eager construction.** With `WithEager`, the new binding is checked for cycles against the existing graph and the resolver is called right away. A cycle, a constructor error or a nil result fails the bind.
5. **Register.** The binding is stored under `bindings[keyType][name]`. Validation happens first, so a rejected bind leaves the container unchanged.

## Resolution

### Explicit resolution

`Resolve(&target)` is `ResolveNamed(&target, "")`. `ResolveNamed`:

1. Requires `target` to be a pointer and uses its element type as the key.
2. Looks up `bindings[elemType][name]`, resolves it, and assigns the result through reflection.
3. **Struct fallback.** If the target is a struct type with no binding, it looks for a binding of the pointer type `*T` and copies the dereferenced value. This lets `var s Service` resolve a `func() *Service` binding.

`binding.resolve` handles the lifetime:

- **Singleton:** takes the binding's mutex, returns `concrete` if it is already set, and otherwise calls the resolver and caches the result. Each binding has its own mutex, so a slow constructor only blocks callers waiting for that same binding.
- **Transient:** calls the resolver every time.

`callResolver` calls the constructor through reflection. If the second return value is a non-nil error, it returns that error. Otherwise, if the first return value is a **nil interface**, it returns `resolver for <T> returned nil`. A nil interface has no dynamic type, so it can't be assigned to the target or passed to another constructor. A nil *pointer* of a concrete type is still a valid result and resolves to `nil`.

### Cycle detection

Before resolving a top-level binding (in `ResolveNamed`, in `ResolveAll` for each binding, and in an eager `bind`), `checkCycles` walks the dependency graph from it by depth-first search. It follows the **unnamed** binding of each parameter type, skips `Lazy[T]` parameters, and skips missing dependencies, which are reported during resolution. If the walk returns to a binding already on the current path, it fails with:

```
circular dependency detected: di.cycleB -> di.cycleC -> di.cycleB
```

The message lists only the bindings in the cycle, not the whole path from the root.

The check runs *before* any binding mutex is taken, and that's what makes it reliable. A check that tracked each goroutine's resolve path as it went wouldn't be enough: two goroutines entering a singleton cycle from opposite ends would each lock one binding's mutex and then wait for the other's, deadlocking before either saw a cycle.

To avoid repeating the walk on every resolve, the container keeps a `generation` counter, incremented by `Clear` and by `bind` (both before the cycle check and after registering). When a binding passes a check, it records the current generation in an atomic `checkedGen` field. The walk is skipped while the two match, so a binding is checked once per change to the graph. Registering a binding can close a cycle, so any change invalidates every earlier check.

The check is static: it reports a cycle even if some bindings in it are singletons that are already built. A cycle in the graph is a configuration error whether or not the instances happen to be cached.

### Constructor arguments

`resolveArguments` goes through the resolver's parameters in order:

- A `Lazy[T]` parameter gets a `Lazy[T]` value holding the container. Nothing is resolved yet.
- Any other parameter is resolved from the **unnamed** binding of its exact type. If that binding is missing, the call fails with `failed resolving argument <type>`.

Named bindings are never injected automatically. You can only reach them with `ResolveNamed` or `ResolveAll`.

### ResolveAll

`ResolveAll(&slice)` resolves every binding, named and unnamed, registered under the slice's element type and stores them in the slice. If there are no bindings it leaves the slice alone and returns `nil`.

## Lazy dependencies

`Lazy[T]` is a struct holding a `*Container`, with a `Resolve() (T, error)` method that calls `Container.Resolve`. Because `T` is resolved only when `Resolve` is called, two services can depend on each other as long as one side takes a `Lazy`.

A `Lazy[T]` parameter is recognised by reflection: any struct type whose name starts with `Lazy[` (see `isLazy`).

## Interface binding (`As[T]`)

### Problem

Go constructors usually return concrete types (`func NewPostgresDB() *postgresDB`), but consumers depend on interfaces (`Database`). Bindings are keyed by the resolver's return type, so binding `NewPostgresDB` directly registers `*postgresDB`. Then a constructor that takes a `Database` can't find it. Before this change, the only workaround was to wrap every constructor in a closure:

```go
di.Bind(func() Database { return NewPostgresDB() })
```

### API

```go
func As[T any]() BindOption                                         // container.go
func BindAs[T any](resolver interface{}, options ...BindOption) error // global.go

c.Bind(NewPostgresDB, di.As[Database]())
di.BindAs[Database](NewPostgresDB)
```

### Why an option, not a new method

Go methods can't take type parameters, so `c.BindAs[T](...)` can't be written. Several shapes were possible:

| Option | Verdict |
| --- | --- |
| `As[T]()` bind option | **Chosen.** Works with every existing `Bind*` method, on the global container and on `*Container`, and combines with `WithName`, `WithTransient` and `WithEager` without new API. |
| Package function `BindAs[T](c *Container, ...)` | Would clash with the global `BindAs` name and doubles the API. |
| Make `Bind` itself generic | Would break every existing caller and the type-inference design. |

`BindAs[T]` is still provided for the global container as a shorthand, since that is where the README examples live.

### Mechanics

- `As[T]` stores `reflect.TypeOf((*T)(nil)).Elem()` in `bindConfig.as`. The pointer trick gets a `reflect.Type` for interface types, which `reflect.TypeOf` can't do from a value.
- In `bind`, the key type becomes `as`. Before that, two checks run:
  - **Assignability:** the return type must be assignable to `as`, otherwise you get `container: <ret> is not assignable to <T>`. `AssignableTo` (not `Implements`) is used so that a non-interface `T` that is identical to the return type also passes.
  - **Self-dependency:** no parameter may be of type `as`. Otherwise resolving `T` would call a constructor that needs `T`.
- Nothing changes on the resolution path. The cached instance is still the concrete value (for example `*postgresDB` boxed in `any`). The reflection operations used to deliver it (`Value.Set`, `Value.Call`, `reflect.Append`) all accept a value assignable to the destination type, and the bind-time check guarantees that.
- Only `T` is registered. The concrete type is **not** also bound, so resolving `*postgresDB` fails unless it is bound separately. Registering both automatically would create two separate singletons for one constructor, which is surprising.

## Global container

`global.go` keeps one package-level `*Container` created by `New()`, and each package function delegates to it. `Clear()` resets it, which tests use to isolate themselves.

## Concurrency model

- `Bind` and `Clear` take the container's **write lock** for the whole operation, including eager construction.
- `Resolve*` take the **read lock** for the whole resolution, including all constructor calls in the dependency graph. Internally, `resolveArguments` reads `bindings` directly because the lock is already held.
- Singleton creation is serialised per binding by `binding.mutex`, so concurrent first resolves of the same type build it only once.
- Cycle checks run under the read lock, before any binding mutex is taken. `generation` only changes under the write lock, so it is stable during a check. `checkedGen` is atomic because several readers may record the same check at once. Since every cycle is rejected before any mutex is locked, binding mutexes can't deadlock through a dependency cycle.

Two things follow from this:

- A constructor **must not call `Bind`** on the same container. It would deadlock waiting for the write lock while its caller holds the read lock.
- Calling `Lazy.Resolve` *inside* a constructor takes the read lock a second time. `sync.RWMutex` read locks aren't reentrant, so this can deadlock if a `Bind` is waiting at the same moment. `Lazy` is meant to be resolved after construction, as in the examples.

## Error handling

| Situation | Behaviour |
| --- | --- |
| Resolver isn't a function | `container: the resolver must be a function` |
| Zero or more than two return values | `need exactly one or two return values` |
| Parameter of the bound type | `can't depend on return type` / `can't depend on bound type` |
| `As[T]` mismatch | `container: <ret> is not assignable to <T>` |
| Dependency cycle not broken by `Lazy` | `circular dependency detected: <A> -> <B> -> <A>` (on `Resolve*`, or on `Bind` when eager) |
| Constructor returns a nil interface | `resolver for <T> returned nil` |
| Missing dependency while building | `failed resolving argument <type>` |
| No binding on `Resolve` | `no binding found for type <T> with name '<name>'` |
| Constructor returns an error | Returned as-is from `Resolve` (or from `Bind` when eager) |

## Known limitations

- **`ResolveAll` order isn't stable.** It iterates over a map, so the order of the returned slice can change between calls.
- **Loose `Lazy` detection.** `isLazy` matches any struct type named `Lazy[...]`, including one defined in another package.
- **Cycles are found late.** A lazy binding that closes a cycle is accepted by `Bind`. The cycle is only reported when something in it is first resolved, the same way missing dependencies are.
- **Errors from building a singleton aren't cached.** If a singleton's constructor fails or returns nil, nothing is cached, so the next resolve calls the constructor again.
- **Missing dependencies show up late.** They are only reported when a lazy binding is first resolved, not at bind time.
