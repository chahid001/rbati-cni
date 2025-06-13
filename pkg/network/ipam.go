package network

import (
	"encoding/json"
	"net"
	"os"
	"fmt"
	"sync"
	"ft_cni/pkg/logs"
)


type IPAM struct {

	subnet 		*net.IPNet // Hold the network
	allocated 	map[string]bool // Check if the IP is taken
	mu 			sync.Mutex // Avoid race condition
	storageFile	string // JSON Backend ipam path

}

func	Inc(ip net.IP) {

	ip = ip.To4()

	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	} 
}

func	GetBroadcast(n *net.IPNet) net.IP {

	if n.IP.To4() == nil {
		logs.Log.Error("The subnet is not an IPV4 network !")
		return nil
	}

	broadcast := make(net.IP, len(n.IP.To4()))

	for i := range n.IP.To4() {
		broadcast[i] = n.IP.To4()[i] | ^n.Mask[i] 
	}

	return broadcast
}



func (i *IPAM) Load() error {

	i.mu.Lock()
	defer i.mu.Unlock()
	
	file, err := os.OpenFile("/var/lib/ipam.json", os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		logs.Log.Error("IPAM: Couldn't open JSON file.", "ERROR", err)
		return err 
	}

	defer file.Close()

	stat, _ := file.Stat()

	if stat.Size() == 0 {
		i.allocated = make(map[string]bool)
		return nil 
	} // Decode The existing data in JSON into map to keep tracking the IPs
	if err := json.NewDecoder(file).Decode(&i.allocated); err != nil {
		logs.Log.Error("IPAM: Couldn't Decode Allocation array to JSON file.", "ERROR", err)
		return err
	}
	return nil
}

func (i *IPAM) Save() error {
	
	file, err := os.Create("/var/lib/ipam.json")

	if err != nil {
		logs.Log.Error("IPAM: Couldn't open JSON file.", "ERROR", err)
		return err 
	}

	defer file.Close()
	// Write all the data inside map to the json
	
	if err := json.NewEncoder(file).Encode(&i.allocated); err != nil {
		logs.Log.Error("IPAM: Couldn't Encode Allocation array to JSON file.", "ERROR", err)
		return err
	}
	return nil
}

/*
	Constructor function to parse the network cidr
	and return the subnet and make an empty map to track it
*/
func NewIPAM(cidr string) (*IPAM, error) {
	_, subnet, err := net.ParseCIDR(cidr)

	if err != nil {
		logs.Log.Error("Couldn't parse Subnet ! Subnet invalid", "ERROR", err)
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

func (i *IPAM) Allocate(candidate string) (net.IP, net.IPMask, error) {
    i.mu.Lock()
    defer i.mu.Unlock()

    mask := i.subnet.Mask

    if candidate != "" {
        if i.allocated[candidate] {
             logs.Log.Error("IP already reserved !")
             return nil, nil, fmt.Errorf("IP address %s is already reserved", candidate)
        }

        newIP := net.ParseIP(candidate)
        if i.subnet.Contains(newIP) { 
            i.allocated[candidate] = true
            err := i.Save()
            if err != nil {
                return nil, nil, fmt.Errorf("failed to save IPAM data: %w", err)
            }
            logs.Log.Info("IP Reserved Succeffully!")
            return newIP, mask, nil 
        } else {
            logs.Log.Error("IP is not part of Subnet !")
            return nil, nil, fmt.Errorf("candidate IP %s is not part of subnet %s", candidate, i.subnet.String())
        }

    } 

	// ------------- Create New IPv4 -----------------
	broadcast := GetBroadcast(i.subnet)
	ip := i.subnet.IP

	Inc(ip)
	
	for ; i.subnet.Contains(ip); Inc(ip) {
        // Skip the broadcast address
        if ip.Equal(broadcast) {
            continue
        }

        candidateStr := ip.String()
        if !i.allocated[candidateStr] {
            i.allocated[candidateStr] = true
            err := i.Save()
            logs.Log.Info("IP Reserved Succeffully!")
            return ip, mask, err
        }
    }



    logs.Log.Error("No IP available to reserve")
    return nil, nil, fmt.Errorf("no available IP addresses in subnet %s", i.subnet.String())
}

func (i *IPAM) Release(ip *net.IP) error {

	i.mu.Lock()
	defer i.mu.Unlock()

	delete(i.allocated, ip.String()) // Delete Ip from the map

	return i.Save()
}