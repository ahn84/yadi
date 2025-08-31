package di_test

import (
	"testing"

	"github.com/ahn84/yadi"
	"github.com/stretchr/testify/require"
)

var constructorCallCount int

type ServiceC struct {
	ServiceD di.Lazy[ServiceD]
}

type ServiceD struct {
	ServiceC *ServiceC
}

type ServiceE struct {
	ServiceF di.Lazy[ServiceF]
}

type ServiceF struct {
	ServiceE *ServiceE
}

func NewServiceE(serviceF di.Lazy[ServiceF]) *ServiceE {
	constructorCallCount++
	return &ServiceE{
		ServiceF: serviceF,
	}
}

func NewServiceF(serviceE *ServiceE) *ServiceF {
	constructorCallCount++
	return &ServiceF{
		ServiceE: serviceE,
	}
}

func TestLazyResolve(t *testing.T) {
	c := di.New()

	err := c.Bind(func(serviceD di.Lazy[ServiceD]) *ServiceC {
		return &ServiceC{
			ServiceD: serviceD,
		}
	})
	require.NoError(t, err)

	err = c.Bind(func(serviceC *ServiceC) *ServiceD {
		return &ServiceD{
			ServiceC: serviceC,
		}
	})
	require.NoError(t, err)

	var serviceC *ServiceC
	err = c.Resolve(&serviceC)
	require.NoError(t, err)
	require.NotNil(t, serviceC)

	serviceD, err := serviceC.ServiceD.Resolve()
	require.NoError(t, err)
	require.NotNil(t, serviceD)

	require.Equal(t, serviceC, serviceD.ServiceC)
}

func TestLazyConstructor(t *testing.T) {
	c := di.New()
	constructorCallCount = 0

	err := c.Bind(NewServiceE)
	require.NoError(t, err)

	err = c.Bind(NewServiceF)
	require.NoError(t, err)

	// At this point, no constructors should have been called (because of lazy binding by default)
	require.Equal(t, 0, constructorCallCount)

	var serviceE *ServiceE
	err = c.Resolve(&serviceE)
	require.NoError(t, err)

	// Only ServiceE constructor should have been called
	require.Equal(t, 1, constructorCallCount)

	_, err = serviceE.ServiceF.Resolve()
	require.NoError(t, err)

	// Now ServiceF constructor should have been called
	require.Equal(t, 2, constructorCallCount)
}
