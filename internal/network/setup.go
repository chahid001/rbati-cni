package network

import (
	"ft_cni/internal/logs"
	"log"

	"github.com/vishvananda/netlink"
)

func SetupVXLAN(name, iface string, vni int) error {

	_, err := netlink.LinkByName(name)
	if err == nil {
		log.Println("[INFO] VXLAN interface already exists")
		return nil
	}


	ifVeth, err := netlink.LinkByName(iface)
	if err != nil {
		return err
	}

	vxlan := &netlink.Vxlan{

		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
		VxlanId:      vni,
		VtepDevIndex: ifVeth.Attrs().Index,
		SrcAddr:      nil, // Funny thing the kernel will take the lead to choose it
		Port:         4789,
		Learning:     false, // To always sent ARP requests to be up to date
		Proxy:        false,
		L2miss:       false,
		L3miss:       false,
	}

	if err := netlink.LinkAdd(vxlan); err != nil {
		logs.Log.Error("Couldn't add VXLAN interface", "ERROR", err)
		return err
	}

	if err := netlink.LinkSetUp(vxlan); err != nil {
		logs.Log.Error("Couldn't set VXLAN interface UP", "ERROR", err)
		return err
	}

	return nil
}
