package main

import (
	"fmt"

	"github.com/ahn84/yadi"
)

type Greeter interface {
	Greet() string
}

type SimpleGreeter struct{}

func (g *SimpleGreeter) Greet() string {
	return "Hello from a concrete constructor!"
}

// NewSimpleGreeter returns a concrete type, as Go constructors usually do.
func NewSimpleGreeter() *SimpleGreeter {
	return &SimpleGreeter{}
}

func main() {
	// Bind the constructor as the Greeter interface
	di.BindAs[Greeter](NewSimpleGreeter)

	var greeter Greeter
	di.Resolve(&greeter)

	fmt.Println(greeter.Greet())
}
