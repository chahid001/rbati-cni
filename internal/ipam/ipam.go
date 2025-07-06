package ipam

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	bolt "go.etcd.io/bbolt"
)

const bucketName = "ipam"

type IPAM struct {
	subnet *net.IPNet
	db     *bolt.DB
	dbPath string
	mu     sync.Mutex
}

func NewIPAM(Subnet, _dbPath string) (*IPAM, error) {
	_, _subnet, err := net.ParseCIDR(Subnet)
	if err != nil {
		log.Printf("[ERROR] Couldn't parse Subnet! Subnet invalid: %v\n", err)
		return nil, err
	}

	if err := os.MkdirAll("/var/lib/cni/rbati-cni", 0755); err != nil {
		return nil, err
	}

	_db, err := bolt.Open(_dbPath, 0600, nil)
	if err != nil {
		log.Printf("[ERROR] Couldn't Open DataBase: %v\n", err)
		return nil, err
	}

	return &IPAM{
		subnet: _subnet,
		db:     _db,
		dbPath: _dbPath,
	}, nil
}

func (i *IPAM) isAllocated(ip string) (bool, error) {
	return i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil
		}
		data := b.Get([]byte(ip))
		if data == nil {
			return nil
		}
		return fmt.Errorf("IP %s is allocated", ip)
	}) == nil, nil
}

func (i *IPAM) setLease(ip string) error {
	return i.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			log.Printf("[ERROR] Couldn't create bucket: %v\n", err)
			return err
		}
		return b.Put([]byte(ip), []byte("1"))
	})
}

func (i *IPAM) Allocate(candidate string) (net.IP, net.IPMask, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	mask := i.subnet.Mask
	ip := i.subnet.IP.Mask(mask)

	if candidate != "" {
		isAllocated, err := i.isAllocated(candidate)
		if err != nil {
			return nil, nil, err
		}

		newIP := net.ParseIP(candidate)
		if !i.subnet.Contains(newIP) {
			log.Println("[ERROR] IP is not part of Subnet !")
			return nil, nil, fmt.Errorf("candidate IP %s is not part of subnet %s", candidate, i.subnet.String())
		}

		if isAllocated {
			log.Printf("[INFO] IP %s already reserved, returning success.\n", candidate)
			return newIP, mask, nil
		}

		if err := i.setLease(candidate); err != nil {
			return nil, nil, err
		}
		return newIP, mask, nil
	}

	for ip := ip.To4(); i.subnet.Contains(ip); incrementIP(ip) {
		lastByte := ip[len(ip)-1]
		if lastByte == 0 || lastByte == 255 {
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

	log.Println("[ERROR] No IP available to reserve")
	return nil, nil, fmt.Errorf("no available IP addresses in subnet %s", i.subnet.String())
}

func (i *IPAM) Release(ip string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(ip))
	})
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