package di_test

import (
	"testing"

	"github.com/ahn84/yadi"
	"github.com/stretchr/testify/require"
)

type Initializable interface {
	Initialize()
}

type ServiceA struct {
	initialized bool
}

func (s *ServiceA) Initialize() {
	s.initialized = true
}

type ServiceB struct {
	initialized bool
}

func (s *ServiceB) Initialize() {
	s.initialized = true
}

func TestResolveAll(t *testing.T) {
	c := di.New()

	err := c.Bind(func() Initializable {
		return &ServiceA{}
	})
	require.NoError(t, err)

	err = c.BindNamed("serviceB", func() Initializable {
		return &ServiceB{}
	})
	require.NoError(t, err)

	var services []Initializable
	err = c.ResolveAll(&services)
	require.NoError(t, err)
	require.Len(t, services, 2)

	for _, s := range services {
		s.Initialize()
	}

	for _, s := range services {
		switch s := s.(type) {
		case *ServiceA:
			require.True(t, s.initialized)
		case *ServiceB:
			require.True(t, s.initialized)
		}
	}
}
