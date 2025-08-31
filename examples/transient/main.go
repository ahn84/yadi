package main

import (
	"fmt"

	"github.com/ahn84/yadi"
)

type Counter interface {
	Increment()
	Count() int
}

type SimpleCounter struct {
	count int
}

func (c *SimpleCounter) Increment() {
	c.count++
}

func (c *SimpleCounter) Count() int {
	return c.count
}

func main() {
	// Singleton by default
	yadi.Bind(func() Counter {
		return &SimpleCounter{}
	})

	// Transient
	yadi.BindNamedTransient("transient", func() Counter {
		return &SimpleCounter{}
	})

	var c1 Counter
	yadi.Resolve(&c1)
	c1.Increment()
	fmt.Printf("Singleton counter 1: %d\n", c1.Count())

	var c2 Counter
	yadi.Resolve(&c2)
	c2.Increment()
	fmt.Printf("Singleton counter 2: %d\n", c2.Count())

	var tc1 Counter
	yadi.ResolveNamed(&tc1, "transient")
	tc1.Increment()
	fmt.Printf("Transient counter 1: %d\n", tc1.Count())

	var tc2 Counter
	yadi.ResolveNamed(&tc2, "transient")
	tc2.Increment()
	fmt.Printf("Transient counter 2: %d\n", tc2.Count())
}

