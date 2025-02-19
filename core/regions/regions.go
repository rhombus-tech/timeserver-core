package regions

import (
	"fmt"
	"sync"
)

// Region represents a geographic region for timeservers
type Region struct {
	ID          string   // Unique identifier for the region
	Name        string   // Human-readable name
	Description string   // Optional description
	Servers     []string // List of server IDs in this region
}

// RegionManager manages server regions and their assignments
type RegionManager struct {
	mu      sync.RWMutex
	regions map[string]*Region // Map of region ID to Region
	// Map of server ID to region ID for quick lookups
	serverRegions map[string]string
}

// NewRegionManager creates a new region manager
func NewRegionManager() *RegionManager {
	return &RegionManager{
		regions:       make(map[string]*Region),
		serverRegions: make(map[string]string),
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
		Servers:     make([]string, 0),
	}

	return nil
}

// RemoveRegion removes a region and unassigns all servers from it
func (rm *RegionManager) RemoveRegion(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	region, exists := rm.regions[id]
	if !exists {
		return fmt.Errorf("region %s does not exist", id)
	}

	// Unassign all servers from this region
	for _, serverID := range region.Servers {
		delete(rm.serverRegions, serverID)
	}

	delete(rm.regions, id)
	return nil
}

// AssignServer assigns a server to a region
func (rm *RegionManager) AssignServer(serverID, regionID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	region, exists := rm.regions[regionID]
	if !exists {
		return fmt.Errorf("region %s does not exist", regionID)
	}

	// Remove server from old region if it exists
	if oldRegionID, exists := rm.serverRegions[serverID]; exists {
		oldRegion := rm.regions[oldRegionID]
		for i, id := range oldRegion.Servers {
			if id == serverID {
				oldRegion.Servers = append(oldRegion.Servers[:i], oldRegion.Servers[i+1:]...)
				break
			}
		}
	}

	// Assign to new region
	region.Servers = append(region.Servers, serverID)
	rm.serverRegions[serverID] = regionID

	return nil
}

// GetServerRegion gets the region ID for a server
func (rm *RegionManager) GetServerRegion(serverID string) (string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	regionID, exists := rm.serverRegions[serverID]
	if !exists {
		return "", fmt.Errorf("server %s not assigned to any region", serverID)
	}

	return regionID, nil
}

// GetRegionServers gets all servers in a region
func (rm *RegionManager) GetRegionServers(regionID string) ([]string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	region, exists := rm.regions[regionID]
	if !exists {
		return nil, fmt.Errorf("region %s does not exist", regionID)
	}

	// Return a copy to prevent external modifications
	servers := make([]string, len(region.Servers))
	copy(servers, region.Servers)

	return servers, nil
}

// ListRegions returns all regions
func (rm *RegionManager) ListRegions() []*Region {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	regions := make([]*Region, 0, len(rm.regions))
	for _, region := range rm.regions {
		regions = append(regions, region)
	}

	return regions
}
