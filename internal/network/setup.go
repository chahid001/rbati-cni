package network

import (
	"log"

	"rbati-cni/internal/ipam"

	"github.com/vishvananda/netlink"
)

func bridgeCleanUP(bridge netlink.Link, ip string, ipam *ipam.IPAM, err error) {

	log.Printf("Couldn't assign IP to Bridge interface; %v\n", err)
	log.Printf("Cleaning.......Deleting Bridge Interafce and releasing IP address")
	if err := netlink.LinkDel(bridge); err != nil {
		log.Printf("Couldn't delete Bridge; %v\n", err)
	}	
	if err := ipam.Release(ip); err != nil {
		log.Printf("Couldn't release IP Address for POD: %v\n", err)
	}
	log.Printf("Released POD IP: %v\n", ip)
	if err != nil {
		log.Printf("Couldn't delete IP; %v\n", err)
	}

}

func SetupBridge(name, ip string) error {

	log.Printf("[INFO] Creating Bridge %v\n", name)
    
	bridge, err := netlink.LinkByName(name)
    if err != nil {

        if _, ok := err.(netlink.LinkNotFoundError); ok {
            log.Printf("[INFO] Bridge %s not found. Creating it.\n", name)
            la := netlink.LinkAttrs{Name: name}
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

	// bridgeIP, bridgeMask, err := ipam.Allocate(ip)
	// if err != nil {
	// 	log.Printf("[ERROR] Couldn't Allocate specified IP for Bridge Interface; %v\n", err)
	// 	log.Printf("[ERROR] Cleaning... Deleting Bridge %v\n", name)
	// 	if err := netlink.LinkDel(bridge); err != nil {
	// 		log.Printf("[ERROR] Couldn't delete Bridge; %v\n", err)
	// 	}
	// 	return err
	// }

	// log.Printf("[INFO] Attached IP Address '%v' to Bridge %v\n", bridgeIP.String() ,name)


	// ones, _ := bridgeMask.Size()
	// bridgeCIDR := fmt.Sprintf("%s/%d", bridgeIP.String(), ones)

	addr, err := netlink.ParseAddr("10.42.0.1/24")
	// if err != nil {
	// 	bridgeCleanUP(bridge, bridgeIP.String(), ipam, err)
	// 	return err
	// }
	log.Println("[INFO] Parsed Address")

	if err := netlink.AddrAdd(bridge, addr); err != nil {
		// bridgeCleanUP(bridge, bridgeIP.String(), ipam, err)
		return err
	}	
	// log.Printf("[INFO] Added IP Address '%v' to Bridge %v\n", bridgeIP.String() ,name)
	
	if err := netlink.LinkSetUp(bridge); err != nil {
		// bridgeCleanUP(bridge, bridgeIP.String(), ipam, err)
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


	vxlan := &netlink.Vxlan{

		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
		VxlanId:      vni,
		VtepDevIndex: ifVeth.Attrs().Index,
		SrcAddr:      addr[0].IP, // I feel betrayed, No the vxlan wont work with NIL
		Port:         4789,
		Learning:     false, // To always sent ARP requests to be up to date
		Proxy:        false,
		L2miss:       false,
		L3miss:       false,
	}

	if err := netlink.LinkAdd(vxlan); err != nil {
		log.Printf("Couldn't add VXLAN interface %v\n", err)
		return err
	}

	if err := netlink.LinkSetUp(vxlan); err != nil {
		log.Printf("Couldn't set VXLAN interface UP %v\n", err)
		if err := netlink.LinkDel(vxlan); err != nil {
			log.Printf("Couldn't delete Bridge %v\n", err)
		}
		return err
	}



	br, err := netlink.LinkByName(bridge)
    if err != nil {
        log.Printf("Couldn't Get Iface %v\n", err)
		return err
    }

    if err := netlink.LinkSetMaster(vxlan, br); err != nil {
        log.Printf("[ERROR] Link bridge to VXLAN %v\n", err)
		return err
    }

	return nil
}