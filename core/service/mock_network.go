package service

import (
	"github.com/rhombus-tech/timeserver-core/core/types"
	"github.com/stretchr/testify/mock"
)

type mockTimeNetwork struct {
	mock.Mock
}

func newMockTimeNetwork() *mockTimeNetwork {
	return &mockTimeNetwork{}
}

func (m *mockTimeNetwork) AddServer(server types.TimeServer) error {
	args := m.Called(server)
	return args.Error(0)
}

func (m *mockTimeNetwork) RemoveServer(serverID string) error {
	args := m.Called(serverID)
	return args.Error(0)
}

func (m *mockTimeNetwork) GetServers() []types.TimeServer {
	args := m.Called()
	if servers := args.Get(0); servers != nil {
		return servers.([]types.TimeServer)
	}
	return nil
}

func (m *mockTimeNetwork) GetServersByRegion(region string) []types.TimeServer {
	args := m.Called(region)
	if servers := args.Get(0); servers != nil {
		return servers.([]types.TimeServer)
	}
	return nil
}

func (m *mockTimeNetwork) GetStatus() (*types.NetworkStatus, error) {
	args := m.Called()
	if status := args.Get(0); status != nil {
		return status.(*types.NetworkStatus), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTimeNetwork) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockTimeNetwork) Stop() error {
	args := m.Called()
	return args.Error(0)
}
