package network

import (
	"crypto/rand"
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

func SetupVXLAN(name, iface, bridge string, vni int) error {

	_, err := netlink.LinkByName(name)
	if err == nil {
		log.Println("[INFO] VXLAN interface already exists")
		return nil
	}
	
	log.Printf("[INFO] Setup VXLAN Based on Interface %v\n", iface)

	ifVeth, err := netlink.LinkByName(iface)
	if err != nil {
		log.Printf("Couldn't Get Iface %v\n", err)
		return err
	}

	log.Printf("[INFO] Looking for default IP on Interface %v\n", iface)
	addr, err := netlink.AddrList(ifVeth, netlink.FAMILY_V4)
	if err != nil {
		log.Printf("Couldn't Get IPs from Iface %v\n", err)
		return err
	}
	log.Printf("[INFO] Found it ,IP %v\n", addr[0].IP.String())


/*
	The kernel create 2 mac addresses, A random One for link/ether 
	that he use on display ip a and for actual L2 trafic.
	And the second is to generate the link-local IPV6 and the one actually 
	reported by my netlink. This made it impossible to have a reliable fdb and arp infra.
	
	This is why i generate my own mac address and i assign it to the interface .
*/

    hwaddr := make(net.HardwareAddr, 6)
    if _, err := rand.Read(hwaddr); err != nil { // Filling it randomly
        log.Printf("Could not generate random MAC address: %v", err)
        return err
    }
	hwaddr[0] |= 0x02 // Setting the MAC to 1 local 

	vxlan := &netlink.Vxlan{

		LinkAttrs: netlink.LinkAttrs{
			Name: name,
			HardwareAddr: hwaddr,
		},
		VxlanId:      vni,
		VtepDevIndex: ifVeth.Attrs().Index,
		SrcAddr:      addr[0].IP, 
		Port:         4789,
		Learning:     false, 
		Proxy:        false,
		L2miss:       false,
		L3miss:       false,
	}

	if err := netlink.LinkAdd(vxlan); err != nil {
		log.Printf("Couldn't add VXLAN interface %v\n", err)
		return err
	}

	podCIDR := "10.42.0.0/32" // Change for later 

	vxlanAddr, err := netlink.ParseAddr(podCIDR)
	if err != nil {
		return err
	}
	log.Println("[INFO] Parsed Address")

	if err := netlink.AddrAdd(vxlan, vxlanAddr); err != nil {
		return err
	}	

	if err := netlink.LinkSetUp(vxlan); err != nil {
		log.Printf("Couldn't set VXLAN interface UP %v\n", err)
		if err := netlink.LinkDel(vxlan); err != nil {
			log.Printf("Couldn't delete Bridge %v\n", err)
		}
		return err
	}

	return nil
}