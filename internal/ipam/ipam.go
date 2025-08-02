package ipam

import (
	"fmt"
	"log"
	"net"
	"os"
	"rbati-cni/internal/types"
	"strconv"
	"sync"

	bolt "go.etcd.io/bbolt"
)

const bucketName = types.BUCKET_NAME

type IPAM struct {
	subnet 		*net.IPNet
	db     		*bolt.DB
	dbPath 		string
	mu     		sync.Mutex
	networkIP	net.IP
    gatewayIP   net.IP 
    broadcastIP net.IP 
}

func NewIPAM(Subnet, _dbPath string) (*IPAM, error) {
	_, _subnet, err := net.ParseCIDR(Subnet)
	if err != nil {
		log.Printf("Couldn't parse Subnet ! Subnet invalid; Error: %v", err)
		return nil, err
	}

	if err := os.MkdirAll(types.IPAM_FODLER_PATH, 0755); err != nil {
		return nil, err
	}

	_db, err := bolt.Open(_dbPath, 0600, nil)
	if err != nil {
		log.Printf("Couldn't Open DataBase; Error: %v", err)
		return nil, err
	}

    _networkIP := _subnet.IP.To4()
    _broadcastIP := make(net.IP, len(_networkIP))
    for i := range _networkIP {
        _broadcastIP[i] = _networkIP[i] | ^_subnet.Mask[i]
    }
    
    // Calculate gateway IP
    _gatewayIP := make(net.IP, len(_networkIP))
    copy(_gatewayIP, _networkIP)
    incrementIP(_gatewayIP)


	return &IPAM{
		subnet: _subnet,
		db:     _db,
		dbPath: _dbPath,
		networkIP: _networkIP,
		gatewayIP: _gatewayIP,
		broadcastIP: _broadcastIP,
	}, nil
}

func (i *IPAM) isAllocated(ip string) (bool, error) {
    var allocated bool
    err := i.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte(bucketName))
        if b == nil {
            // Bucket doesn't exist, so no IPs are allocated yet
            allocated = false
            return nil
        }
        // If Get returns a value, the IP is allocated
        if b.Get([]byte(ip)) != nil {
            allocated = true
        } else {
            allocated = false
        }
        return nil
    })

    if err != nil {
        return false, err // Return any error from accessing the database
    }

    return allocated, nil
}

func (i *IPAM) setLease(ip string) error {
	return i.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			log.Printf("Couldn't Check Allocation; Error: %v", err)
			return err
		}
		if err := b.Put([]byte(ip), []byte("1")); err != nil {
			return err
		}

		i.CountAllocatedIPs()

		return nil
	})
}

func (i *IPAM) checkCandidate(candidate net.IP) bool {

	if candidate.To4().Equal(i.networkIP) || 
		candidate.To4().Equal(net.IP(i.gatewayIP)) ||
			candidate.To4().Equal(i.broadcastIP) {
		return false
	}

	return true
}

func (i *IPAM) getCandidate() (net.IP, net.IPMask, error) {

	mask := i.subnet.Mask
	ip := i.subnet.IP.Mask(mask)
	
	for ip := ip.To4(); i.subnet.Contains(ip); incrementIP(ip) {

		if !i.checkCandidate(ip) {
			continue
		}

		ipStr := ip.String()
		isAllocated, err := i.isAllocated(ipStr)
		if err != nil {
			return nil, nil, err
		}
		if isAllocated {
			continue
		}
		err = i.setLease(ipStr)
		if err != nil {
			return nil, nil, err
		}
		return ip, mask, nil
	}

	return nil, nil, fmt.Errorf("No IP available to reserve")

}

func (i *IPAM) Allocate(candidate string) (net.IP, net.IPMask, error) {
    i.mu.Lock()
    defer i.mu.Unlock()

    mask := i.subnet.Mask

    if candidate != "" {
        isAllocated, err := i.isAllocated(candidate)
        if err != nil {
            return nil, nil, err
        }

        newIP := net.ParseIP(candidate)
        if newIP == nil {
            return nil, nil, fmt.Errorf("candidate IP %s is not a valid IP address", candidate)
        }

        if !i.subnet.Contains(newIP) {
            return nil, nil, fmt.Errorf("candidate IP %s is not part of subnet %s", candidate, i.subnet.String())
        }
        
        if isAllocated {
            log.Printf("[INFO] IP %s already reserved, returning success for idempotency.", candidate)
            return newIP, mask, nil 
        }
        
        if !i.checkCandidate(newIP) {
            return nil, nil, fmt.Errorf("candidate IP %s is a reserved network address (network, gateway, or broadcast)", candidate)
        }

        if err := i.setLease(candidate); err != nil {
            return nil, nil, err
        }
        return newIP, mask, nil
    }

    return i.getCandidate()
}

func (i *IPAM) Release(ip string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil
		}
		if err := b.Delete([]byte(ip)); err != nil {
			return err
		}

		i.CountAllocatedIPs()

		return nil
	})
}

func (i *IPAM) CountAllocatedIPs() {

	i.mu.Lock()

	var count int

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b != nil {
			count = b.Stats().KeyN
		}

		return nil
	})
	if err != nil {
		log.Printf("[ERROR] Couldn't read from DB: %v", err)
	}

	countStr := strconv.Itoa(count)
	if err := os.WriteFile(types.COUNT_FILEPATH, []byte(countStr), 0644); err != nil {
		log.Printf("[ERROR] Couldn't write file: %v", err)
	}

}


func (i *IPAM) Close() {
	i.db.Close()
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}
}