package openstack

import "context"

type MockClient struct {
    Created []string
}

func (m *MockClient) CreateServer(ctx context.Context, name, imageID, flavorID, userdata string, networkIDs []string, meta map[string]string) (string, error) {
    m.Created = append(m.Created, name)
    return "mock-server-id", nil
}

func (m *MockClient) GetServer(ctx context.Context, id string) (ServerInfo, error) {
    return ServerInfo{
        ID:     id,
        Name:   "mock-server",
        Status: "ACTIVE",
        IPs:    []string{"10.0.0.10"},
        CPU:    2,
        Memory: 4 * 1024 * 1024 * 1024,
    }, nil
}

func (m *MockClient) DeleteServer(ctx context.Context, id string) error {
    return nil
}
