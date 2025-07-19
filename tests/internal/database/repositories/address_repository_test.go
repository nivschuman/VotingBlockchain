package repositories_test

import (
	"net"
	"testing"
	"time"

	repositories "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	networking_models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	inits "github.com/nivschuman/VotingBlockchain/tests/init"
)

func TestGetAddresses(t *testing.T) {
	inits.ResetTestDatabase()

	now := time.Now()
	old := now.Add(-10 * time.Minute)

	addresses := []*networking_models.Address{
		{Ip: net.ParseIP("192.168.1.1"), Port: 8333, NodeType: 1},
		{Ip: net.ParseIP("10.0.0.2"), Port: 8333, NodeType: 2},
		{Ip: net.ParseIP("172.16.0.3"), Port: 18333, NodeType: 1},
		{Ip: net.ParseIP("::1"), Port: 8333, NodeType: 3},
		{Ip: net.ParseIP("2001:db8::1"), Port: 8334, NodeType: 1},
	}

	for idx, address := range addresses {
		err := repositories.GlobalAddressRepository.InsertIfNotExists(address)
		if err != nil {
			t.Fatalf("failed to insert test address: %v", err)
		}

		var seenTime *time.Time
		if idx%2 == 0 {
			seenTime = &now
		} else {
			seenTime = &old
		}

		err = repositories.GlobalAddressRepository.UpdateLastSeen(address, seenTime)
		if err != nil {
			t.Fatalf("failed to update last seen: %v", err)
		}
	}

	excludeIPs := []net.IP{net.ParseIP("192.168.1.1")}
	excludePorts := []uint16{8334}

	addresses, err := repositories.GlobalAddressRepository.GetAddresses(10, excludeIPs, excludePorts)
	if err != nil {
		t.Fatalf("failed to get addresses: %v", err)
	}

	for _, addr := range addresses {
		if addr.Ip.String() == "192.168.1.1" {
			t.Fatalf("excluded ip was returned")
		}

		if addr.Port == uint16(8334) {
			t.Fatalf("excluded port was returned")
		}
	}

	for _, a := range addresses {
		t.Logf("Returned: %s:%d", a.Ip.String(), a.Port)
	}
}
