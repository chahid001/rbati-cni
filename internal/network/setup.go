package network

import (
	"log"

	"github.com/vishvananda/netlink"
)

func SetupVXLAN(name, iface string, vni int) error {

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
		return err
	}

	return nil
}
