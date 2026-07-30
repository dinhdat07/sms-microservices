package mock

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockReportingService struct {
	mock.Mock
}

func NewMockReportingService(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockReportingService {
	mock := &MockReportingService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

func (m *MockReportingService) RequestReport(ctx context.Context, email string, startDate string, endDate string) error {
	args := m.Called(ctx, email, startDate, endDate)
	return args.Error(0)
}

func (m *MockReportingService) ExecuteDailyMaintenance(ctx context.Context, startTime time.Time, endTime time.Time) error {
	args := m.Called(ctx, startTime, endTime)
	return args.Error(0)
}
