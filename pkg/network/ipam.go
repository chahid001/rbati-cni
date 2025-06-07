package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)


type IPAM struct {

	subnet 		*net.IPNet // Hold the network
	allocated 	map[string]bool // Check if the IP is taken
	mu 			sync.Mutex // Avoid race condition
	storageFile	string // JSON Backend ipam path

}


func (i *IPAM) Load() error {

	i.mu.Lock()
	defer i.mu.Unlock()
	
	file, err := os.OpenFile("/var/lib/ipam.json", os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		return err 
	}

	defer file.Close()

	stat, _ := file.Stat()

	if stat.Size() == 0 {
		i.allocated = make(map[string]bool)
		return nil 
	} // Decode The existing data in JSON into map to keep tracking the IPs
	return json.NewDecoder(file).Decode(&i.allocated)
}

func (i *IPAM) Save() error {
	
	file, err := os.Create("/var/lib/ipam.json")

	if err != nil {
		return err
	}

	defer file.Close()
	// Write all the data inside map to the json
	return json.NewEncoder(file).Encode(i.allocated)
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

	path := "/var/lib/ipam.json"

	ipam := &IPAM{
		subnet: subnet,
		allocated: make(map[string]bool),
		storageFile: path,
	}

	err = ipam.Load()

	return ipam, err
}

func (i *IPAM) Allocate() (net.IP, net.IPMask ,error) {
	
	i.mu.Lock()
	defer i.mu.Unlock()

	ip := i.subnet.IP.To4()
	mask := i.subnet.Mask

	for host:= 2; host < 255; host++ {
		
		candidate := net.IPv4(ip[0], ip[1], ip[2], byte(host)).String()

		if !i.allocated[candidate] {
			i.allocated[candidate] = true
			err := i.Save()
			return net.ParseIP(candidate), mask, err 
		}
	}

	return nil, nil, fmt.Errorf("No ip")
}

func (i *IPAM) Release(ip *net.IP) error {

	i.mu.Lock()
	defer i.mu.Unlock()

	delete(i.allocated, ip.String()) // Delete Ip from the map

	return i.Save()
}