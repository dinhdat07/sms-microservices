package mock

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockRawUptimeProvider struct {
	mock.Mock
}

func NewMockRawUptimeProvider(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockRawUptimeProvider {
	mock := &MockRawUptimeProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

func (m *MockRawUptimeProvider) CalculateRawUptimeStats(ctx context.Context, startTime time.Time, endTime time.Time) (int64, int64, error) {
	args := m.Called(ctx, startTime, endTime)
	return args.Get(0).(int64), args.Get(1).(int64), args.Error(2)
}

func (m *MockRawUptimeProvider) CleanupOldData(ctx context.Context, olderThan time.Time) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}
