package network

import (
	"ft_cni/pkg/logs"
	"net"

	"github.com/vishvananda/netlink"
)


func SetupVXLAN(name string, vni int, localIP, remoteIP net.IP) error {

	vxlan := &netlink.Vxlan{

		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
		VxlanId: vni,
		VtepDevIndex: 3,
		Group: remoteIP,
		SrcAddr: localIP,
		Port: 4789,
		Learning: false,
		Proxy: false,
		L2miss: false,
		L3miss: false,
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