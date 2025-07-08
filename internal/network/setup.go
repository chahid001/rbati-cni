package network

import (
	"fmt"
	"log"
	"net"

	"github.com/vishvananda/netlink"
)

func bridgeCleanUP(bridge netlink.Link, err error) {

	log.Printf("Couldn't assign IP to Bridge interface; %v\n", err)
	log.Printf("Cleaning.......Deleting Bridge Interafce")
	if err := netlink.LinkDel(bridge); err != nil {
		log.Printf("Couldn't delete Bridge; %v\n", err)
	}	
}

func SetupBridge(name, ip string) error {

	log.Printf("[INFO] Creating Bridge %v\n", name)
    
	bridge, err := netlink.LinkByName(name)
    if err != nil {

        if _, ok := err.(netlink.LinkNotFoundError); ok {
            log.Printf("[INFO] Bridge %s not found. Creating it.\n", name)
            la := netlink.LinkAttrs{Name: name, MTU: 1450}
            newBridge := &netlink.Bridge{LinkAttrs: la}
            if err := netlink.LinkAdd(newBridge); err != nil {
                log.Printf("[ERROR] Could not create bridge %s: %v\n", name, err)
                return err
            }
            bridge = newBridge 
        } else {
            log.Printf("[ERROR] Failed to find bridge %s: %v\n", name, err)
            return err
        }
    } else {
        log.Printf("[INFO] Bridge %s already exists. Using existing device.\n", name)
    }

	bridgeIP := net.ParseIP(ip)
	// bridgeMask := bridgeIP.DefaultMask()
	
	// ones, _ := bridgeMask.Size()

	bridgeCIDR := fmt.Sprintf("%s/24", bridgeIP.String())

	addr, err := netlink.ParseAddr(bridgeCIDR)
	if err != nil {
		bridgeCleanUP(bridge, err)
		return err
	}
	log.Println("[INFO] Parsed Address")

	if err := netlink.AddrAdd(bridge, addr); err != nil {
		bridgeCleanUP(bridge, err)
		return err
	}	
	log.Printf("[INFO] Added IP Address '%v' to Bridge %v\n", bridgeIP.String() ,name)
	
	if err := netlink.LinkSetUp(bridge); err != nil {
		bridgeCleanUP(bridge, err)
		return err
	}
	log.Printf("[INFO] Set Bridge %v UP\n", name)
	
	return nil  
}
