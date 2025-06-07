package ipam

import (
	"net"
	"sync"
	"fmt"
)


type IPAM struct {

	subnet *net.IPNet // Hold the network
	allocated map[string]bool // Check if the IP is taken
	mu sync.Mutex // Avoid race condition

}


/*
	Constructor function to parse the network cidr
	and return the subnet and make an empty map to track it
*/
func NewIPAM(cidr string) (*IPAM, error) {
	_, subnet, err := net.ParseCIDR(cidr)

	if err != nil {
		return nil, err
	}

	return &IPAM{
		subnet: subnet,
		allocated: make(map[string]bool),
	}, nil
}

func (i *IPAM) Allocate() (net.IP, net.IPMask ,error) {
	
	i.mu.Lock()
	defer i.mu.Unlock()

	ip := i.subnet.IP.To4()
	mask := i.subnet.Mask

	for host:= 1; host < 255; host++ {
		
		candidate := net.IPv4(ip[0], ip[1], ip[2], byte(host))

		if !i.allocated[candidate.String()] && i.subnet.Contains(candidate) {
			i.allocated[candidate.String()] = true
			return candidate, mask, nil
		}
	}

	return nil, nil, fmt.Errorf("No ip")
}

func (i *IPAM) Release(ip *net.IP) {

	i.mu.Lock()
	defer i.mu.Unlock()

	delete(i.allocated, ip.String()) // Delete Ip from the map

}