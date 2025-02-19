package regions

import (
	"fmt"
	"testing"
)

func TestRegionManager(t *testing.T) {
	rm := NewRegionManager()

	// Test adding regions
	t.Run("AddRegion", func(t *testing.T) {
		// Add valid region
		err := rm.AddRegion("us-east", "US East", "East Coast Region")
		if err != nil {
			t.Errorf("Failed to add region: %v", err)
		}

		// Try adding duplicate region
		err = rm.AddRegion("us-east", "US East 2", "Duplicate")
		if err == nil {
			t.Error("Expected error when adding duplicate region")
		}

		// Add another region
		err = rm.AddRegion("us-west", "US West", "West Coast Region")
		if err != nil {
			t.Errorf("Failed to add second region: %v", err)
		}
	})

	// Test assigning servers
	t.Run("AssignServer", func(t *testing.T) {
		// Assign server to valid region
		err := rm.AssignServer("server1", "us-east")
		if err != nil {
			t.Errorf("Failed to assign server: %v", err)
		}

		// Try assigning to non-existent region
		err = rm.AssignServer("server2", "invalid-region")
		if err == nil {
			t.Error("Expected error when assigning to non-existent region")
		}

		// Assign server to different region
		err = rm.AssignServer("server1", "us-west")
		if err != nil {
			t.Errorf("Failed to reassign server: %v", err)
		}

		// Verify server is in new region
		regionID, err := rm.GetServerRegion("server1")
		if err != nil {
			t.Errorf("Failed to get server region: %v", err)
		}
		if regionID != "us-west" {
			t.Errorf("Expected server in us-west, got %s", regionID)
		}
	})

	// Test getting region servers
	t.Run("GetRegionServers", func(t *testing.T) {
		// Add more servers
		rm.AssignServer("server2", "us-west")
		rm.AssignServer("server3", "us-west")

		servers, err := rm.GetRegionServers("us-west")
		if err != nil {
			t.Errorf("Failed to get region servers: %v", err)
		}

		if len(servers) != 3 {
			t.Errorf("Expected 3 servers, got %d", len(servers))
		}

		// Try getting servers from non-existent region
		_, err = rm.GetRegionServers("invalid-region")
		if err == nil {
			t.Error("Expected error when getting servers from non-existent region")
		}
	})

	// Test removing regions
	t.Run("RemoveRegion", func(t *testing.T) {
		// Remove existing region
		err := rm.RemoveRegion("us-east")
		if err != nil {
			t.Errorf("Failed to remove region: %v", err)
		}

		// Try removing non-existent region
		err = rm.RemoveRegion("invalid-region")
		if err == nil {
			t.Error("Expected error when removing non-existent region")
		}

		// Verify servers from removed region are unassigned
		_, err = rm.GetServerRegion("server1")
		if err != nil {
			t.Error("Server should still exist in us-west")
		}
	})

	// Test listing regions
	t.Run("ListRegions", func(t *testing.T) {
		regions := rm.ListRegions()
		if len(regions) != 1 {
			t.Errorf("Expected 1 region, got %d", len(regions))
		}

		if regions[0].ID != "us-west" {
			t.Errorf("Expected us-west region, got %s", regions[0].ID)
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	rm := NewRegionManager()
	rm.AddRegion("us-east", "US East", "East Coast Region")

	// Test concurrent server assignments
	t.Run("ConcurrentAssignments", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				serverID := fmt.Sprintf("server%d", id)
				err := rm.AssignServer(serverID, "us-east")
				if err != nil {
					t.Errorf("Failed concurrent assignment: %v", err)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all servers were assigned
		servers, _ := rm.GetRegionServers("us-east")
		if len(servers) != 10 {
			t.Errorf("Expected 10 servers, got %d", len(servers))
		}
	})
}
