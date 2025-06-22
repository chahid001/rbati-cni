package network

import (
	"encoding/json"
	"fmt"
	"ft_cni/internal/logs"
	"ft_cni/internal/types"
	"net"
	"os"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

const bucketName = "ipam"

type IPAM struct {
	subnet   *net.IPNet // Hold the network
	db       *bolt.DB
	dbPath   string
	mu       sync.Mutex // Avoid race condition
	leaseTTL int64      // in seconds

}

/*
Constructor function to parse the network cidr
and return the subnet and make an empty map to track it
*/
func NewIPAM(Subnet, _dbPath string, ttlSeconds int64) (*IPAM, error) {
	_, _subnet, err := net.ParseCIDR(Subnet)

	if err != nil {
		logs.Log.Error("Couldn't parse Subnet ! Subnet invalid", "ERROR", err)
		return nil, err
	}

	if err := os.MkdirAll("/var/lib/cni/ftcni", 0755); err != nil {
		return nil, err
	}

	_db, err := bolt.Open(_dbPath, 0600, nil)

	if err != nil {
		logs.Log.Error("Couldn't Open DataBase", "ERROR", err)
		return nil, err
	}

	return &IPAM{
		subnet:   _subnet,
		db:       _db,
		dbPath:   _dbPath,
		leaseTTL: ttlSeconds,
	}, nil

}

func (i *IPAM) isAllocated(ip string) (bool, error) {

	var lease types.LeaseInfo

	err := i.db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil // Bucket is empty -> no leases
		}

		data := b.Get([]byte(ip))
		if data == nil {
			return nil // No IP is allocated
		}

		return json.Unmarshal(data, &lease)
	})

	if err != nil {
		logs.Log.Error("Couldn't Check Allocation", "ERROR", err)
		return false, err
	}

	return lease.TimeStamp != 0, nil
}

func (i *IPAM) setLease(ip, _clientID string) error {

	return i.db.Update(func(tx *bolt.Tx) error {

		b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			logs.Log.Error("Couldn't Check Allocation", "ERROR", err)
			return err
		}

		lease := types.LeaseInfo{
			TimeStamp: time.Now().Unix(),
			ClientID:  _clientID,
		}

		data, err := json.Marshal(lease)
		if err != nil {
			logs.Log.Error("Couldn't Marshal", "ERROR", err)
			return err
		}

		return b.Put([]byte(ip), data)
	})
}

func (i *IPAM) Allocate(candidate, _clientID string) (net.IP, net.IPMask, error) {

	i.mu.Lock()
	defer i.mu.Unlock()

	i.CleanExpired()

	mask := i.subnet.Mask
	ip := i.subnet.IP.Mask(mask)

	// ------------- Create Custom IPv4 -----------------
	if candidate != "" {
		isAllocated, err := i.isAllocated(candidate)

		if isAllocated {
			logs.Log.Error("IP already reserved !")
			return nil, nil, fmt.Errorf("IP address %s is already reserved", candidate)
		}
		if err != nil {
			return nil, nil, err
		}

		newIP := net.ParseIP(candidate)
		if i.subnet.Contains(newIP) {
			err := i.setLease(candidate, _clientID)
			if err != nil {
				return nil, nil, err
			}
		} else {
			logs.Log.Error("IP is not part of Subnet !")
			return nil, nil, fmt.Errorf("candidate IP %s is not part of subnet %s", candidate, i.subnet.String())
		}

	}

	// ------------- Create New IPv4 -----------------

	for ip := ip.To4(); i.subnet.Contains(ip); incrementIP(ip) {

		ipStr := ip.String()

		isAllocated, err := i.isAllocated(ipStr)
		if err != nil {
			return nil, nil, err
		}

		if isAllocated {
			continue
		}

		err = i.setLease(ipStr, _clientID)

		if err != nil {
			return nil, nil, err
		}

		return ip, mask, nil

	}

	logs.Log.Error("No IP available to reserve")
	return nil, nil, fmt.Errorf("no available IP addresses in subnet %s", i.subnet.String())
}

func (i *IPAM) Release(ip string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil // If bucket doesn't exist, nothing to do
		}
		return b.Delete([]byte(ip))
	})
}

func (i *IPAM) CleanExpired() error {
	now := time.Now().Unix()

	return i.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil
		}

		// Go through all keys (IPs) in the bucket
		return b.ForEach(func(k, v []byte) error {
			var lease types.LeaseInfo

			// Deserialize the JSON lease info
			if err := json.Unmarshal(v, &lease); err != nil {
				return err
			}

			// Check expiration: if too old, delete it
			if now-lease.TimeStamp > i.leaseTTL {
				return b.Delete(k)
			}
			return nil
		})
	})
}

func (i *IPAM) Close() {
	i.db.Close()
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break // stop if we didn’t overflow the byte
		}
	}
}
