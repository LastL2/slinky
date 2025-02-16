// Code generated by mockery v2.40.3. DO NOT EDIT.

package mocks

import (
	metrics "github.com/skip-mev/slinky/providers/base/metrics"
	mock "github.com/stretchr/testify/mock"

	types "github.com/skip-mev/slinky/providers/types"
)

// ProviderMetrics is an autogenerated mock type for the ProviderMetrics type
type ProviderMetrics struct {
	mock.Mock
}

// AddProviderResponse provides a mock function with given fields: providerName, status, providerType
func (_m *ProviderMetrics) AddProviderResponse(providerName string, status metrics.Status, providerType types.ProviderType) {
	_m.Called(providerName, status, providerType)
}

// AddProviderResponseByID provides a mock function with given fields: providerName, id, status, providerType
func (_m *ProviderMetrics) AddProviderResponseByID(providerName string, id string, status metrics.Status, providerType types.ProviderType) {
	_m.Called(providerName, id, status, providerType)
}

// LastUpdated provides a mock function with given fields: providerName, id, providerType
func (_m *ProviderMetrics) LastUpdated(providerName string, id string, providerType types.ProviderType) {
	_m.Called(providerName, id, providerType)
}

// NewProviderMetrics creates a new instance of ProviderMetrics. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewProviderMetrics(t interface {
	mock.TestingT
	Cleanup(func())
}) *ProviderMetrics {
	mock := &ProviderMetrics{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
