package main

import (
	"fmt"

	"github.com/ahn84/yadi"
)

type Initializable interface {
	Initialize()
}

type ServiceA struct{}

func (s *ServiceA) Initialize() {
	fmt.Println("ServiceA initialized")
}

type ServiceB struct{}

func (s *ServiceB) Initialize() {
	fmt.Println("ServiceB initialized")
}

func main() {
	di.Bind(func() Initializable {
		return &ServiceA{}
	})
	di.BindNamed("b", func() Initializable {
		return &ServiceB{}
	})

	var services []Initializable
	di.ResolveAll(&services)

	for _, s := range services {
		s.Initialize()
	}
}
