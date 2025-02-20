package regions

import (
	"fmt"
	"sync"
)

// Region represents a geographic region
type Region struct {
	ID          string
	Name        string
	Description string
}

// RegionManager manages regions and server assignments
type RegionManager struct {
	regions      map[string]*Region
	serverRegion map[string]string
	mu          sync.RWMutex
}

// NewRegionManager creates a new region manager
func NewRegionManager() *RegionManager {
	return &RegionManager{
		regions:      make(map[string]*Region),
		serverRegion: make(map[string]string),
	}
}

// AddRegion adds a new region
func (rm *RegionManager) AddRegion(id, name, description string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.regions[id]; exists {
		return fmt.Errorf("region %s already exists", id)
	}

	rm.regions[id] = &Region{
		ID:          id,
		Name:        name,
		Description: description,
	}

	return nil
}

// RemoveRegion removes a region and all its servers
func (rm *RegionManager) RemoveRegion(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.regions[id]; !exists {
		return fmt.Errorf("region %s not found", id)
	}

	// Remove all servers in the region
	for serverID, regionID := range rm.serverRegion {
		if regionID == id {
			delete(rm.serverRegion, serverID)
		}
	}

	delete(rm.regions, id)
	return nil
}

// GetRegion gets a region by ID
func (rm *RegionManager) GetRegion(id string) (*Region, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	region, exists := rm.regions[id]
	if !exists {
		return nil, fmt.Errorf("region %s not found", id)
	}

	return region, nil
}

// AssignServer assigns a server to a region
func (rm *RegionManager) AssignServer(serverID, regionID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.regions[regionID]; !exists {
		return fmt.Errorf("region %s not found", regionID)
	}

	rm.serverRegion[serverID] = regionID
	return nil
}

// GetServerRegion gets the region ID for a server
func (rm *RegionManager) GetServerRegion(serverID string) (string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	regionID, exists := rm.serverRegion[serverID]
	if !exists {
		return "", fmt.Errorf("server %s not assigned to any region", serverID)
	}

	return regionID, nil
}

// GetRegionServers gets all servers in a region
func (rm *RegionManager) GetRegionServers(regionID string) ([]string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if _, exists := rm.regions[regionID]; !exists {
		return nil, fmt.Errorf("region %s not found", regionID)
	}

	servers := make([]string, 0)
	for serverID, sRegionID := range rm.serverRegion {
		if sRegionID == regionID {
			servers = append(servers, serverID)
		}
	}

	return servers, nil
}

// ListRegions returns a list of all regions
func (rm *RegionManager) ListRegions() []*Region {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	regions := make([]*Region, 0, len(rm.regions))
	for _, region := range rm.regions {
		regions = append(regions, region)
	}

	return regions
}

// ListServersInRegion returns a list of server IDs in a region
func (rm *RegionManager) ListServersInRegion(regionID string) ([]string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if _, exists := rm.regions[regionID]; !exists {
		return nil, fmt.Errorf("region %s not found", regionID)
	}

	servers := make([]string, 0)
	for serverID, sRegionID := range rm.serverRegion {
		if sRegionID == regionID {
			servers = append(servers, serverID)
		}
	}

	return servers, nil
}
