package service

import (
	"context"
	"time"

	"github.com/rhombus-tech/timeserver-core/core/types"
)

// MockFastPathHandler implements FastPathHandler for testing
type MockFastPathHandler struct {
	timestamp *types.SignedTimestamp
	err       error
}

func NewMockFastPathHandler() *MockFastPathHandler {
	return &MockFastPathHandler{
		timestamp: &types.SignedTimestamp{
			Time:     time.Now(),
			ServerID: "mock-server",
			RegionID: "mock-region",
		},
	}
}

func (m *MockFastPathHandler) GetTimestamp(ctx context.Context) (*types.SignedTimestamp, error) {
	return m.timestamp, m.err
}

func (m *MockFastPathHandler) SetTimestamp(ts *types.SignedTimestamp) {
	m.timestamp = ts
}

func (m *MockFastPathHandler) SetError(err error) {
	m.err = err
}
