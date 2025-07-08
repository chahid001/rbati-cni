package network

import (
	"fmt"
	"log"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
)

type PeerInfo struct {
	Name   string
	IP     net.IP
	Subnet *net.IPNet
}

func EnsureVXLANInterface(name, iface string, vni int) (*netlink.Vxlan, error) {
	_, err := netlink.LinkByName(name)
	if err == nil {
		log.Printf("[INFO] VXLAN interface %s already exists", name)
		vx, _ := netlink.LinkByName(name)
		return vx.(*netlink.Vxlan), nil
	}

	ifVeth, err := netlink.LinkByName(iface)
	if err != nil {
		return nil, fmt.Errorf("cannot find underlying interface %s: %w", iface, err)
	}

	addrList, err := netlink.AddrList(ifVeth, netlink.FAMILY_V4)
	if err != nil || len(addrList) == 0 {
		return nil, fmt.Errorf("cannot get IP from %s: %w", iface, err)
	}

	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
			MTU:  1450,
		},
		VxlanId:      vni,
		VtepDevIndex: ifVeth.Attrs().Index,
		SrcAddr:      addrList[0].IP,
		Port:         4789,
		Learning:     false,
		Proxy:        false,
		L2miss:       false,
		L3miss:       false,
	}

	if err := netlink.LinkAdd(vxlan); err != nil {
		return nil, fmt.Errorf("failed to create vxlan interface: %w", err)
	}

	if err := netlink.LinkSetUp(vxlan); err != nil {
		netlink.LinkDel(vxlan)
		return nil, fmt.Errorf("failed to bring up vxlan: %w", err)
	}

	return vxlan, nil
}

func UpdateFDBAndRoutes(vx *netlink.Vxlan, peers []PeerInfo, myNodeName string) error {
	for _, peer := range peers {
		if peer.Name == myNodeName {
			continue
		}

		mac := MacForSubnet(peer.Subnet)

		log.Printf("[INFO] FDB entry: MAC %s -> IP %s", mac, peer.IP)
		neigh := &netlink.Neigh{
			LinkIndex:    vx.Index,
			Family:       syscall.AF_BRIDGE,
			State:        netlink.NUD_PERMANENT,
			HardwareAddr: mac,
			IP:           peer.IP,
		}
		if err := netlink.NeighSet(neigh); err != nil {
			log.Printf("[WARN] Failed to add FDB entry for %s: %v", peer.Name, err)
		}

		gw := make(net.IP, len(peer.Subnet.IP))
		copy(gw, peer.Subnet.IP)
		gw[len(gw)-1]++

		log.Printf("[INFO] ARP entry: IP %s -> MAC %s", gw, mac)
		arp := &netlink.Neigh{
			LinkIndex:    vx.Index,
			Family:       syscall.AF_INET,
			State:        netlink.NUD_PERMANENT,
			IP:           gw,
			HardwareAddr: mac,
		}
		if err := netlink.NeighSet(arp); err != nil {
			log.Printf("[WARN] Failed to add ARP entry for %s: %v", peer.Name, err)
		}

		log.Printf("[INFO] Route: Subnet %s via %s", peer.Subnet, gw)
		route := &netlink.Route{
			LinkIndex: vx.Index,
			Dst:       peer.Subnet,
			Gw:        gw,
		}
		if err := netlink.RouteReplace(route); err != nil {
			log.Printf("[WARN] Failed to add route for %s: %v", peer.Name, err)
		}
	}
	return nil
}

func MacForSubnet(subnet *net.IPNet) net.HardwareAddr {
	ip := subnet.IP.To4()
	return net.HardwareAddr{0x0a, 0x58, ip[0], ip[1], ip[2], ip[3]}
}

// package network

// import (
// 	"fmt"
// 	"log"
// 	"net"

// 	"github.com/vishvananda/netlink"
// )

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

// func SetupVXLAN(name, iface, bridge string, vni int) error {

// 	_, err := netlink.LinkByName(name)
// 	if err == nil {
// 		log.Println("[INFO] VXLAN interface already exists")
// 		return nil
// 	}
	
// 	log.Printf("[INFO] Setup VXLAN Based on Interface %v\n", iface)

// 	ifVeth, err := netlink.LinkByName(iface)
// 	if err != nil {
// 		log.Printf("Couldn't Get Iface %v\n", err)
// 		return err
// 	}

// 	log.Printf("[INFO] Looking for default IP on Interface %v\n", iface)
// 	addr, err := netlink.AddrList(ifVeth, netlink.FAMILY_V4)
// 	if err != nil {
// 		log.Printf("Couldn't Get IPs from Iface %v\n", err)
// 		return err
// 	}
// 	log.Printf("[INFO] Found it ,IP %v\n", addr[0].IP.String())


// 	vxlan := &netlink.Vxlan{

// 		LinkAttrs: netlink.LinkAttrs{
// 			Name: name,
// 			MTU: 1450,
// 		},
// 		VxlanId:      vni,
// 		VtepDevIndex: ifVeth.Attrs().Index,
// 		SrcAddr:      addr[0].IP, // I feel betrayed, No the vxlan wont work with NIL
// 		Port:         4789,
// 		Learning:     false, // To always sent ARP requests to be up to date
// 		Proxy:        false,
// 		L2miss:       false,
// 		L3miss:       false,
// 	}

// 	if err := netlink.LinkAdd(vxlan); err != nil {
// 		log.Printf("Couldn't add VXLAN interface %v\n", err)
// 		return err
// 	}

// 	if err := netlink.LinkSetUp(vxlan); err != nil {
// 		log.Printf("Couldn't set VXLAN interface UP %v\n", err)
// 		if err := netlink.LinkDel(vxlan); err != nil {
// 			log.Printf("Couldn't delete Bridge %v\n", err)
// 		}
// 		return err
// 	}


// 	return nil
// }