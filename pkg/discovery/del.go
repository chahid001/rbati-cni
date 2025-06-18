package discovery

import (
	"ft_cni/pkg/load"
	"ft_cni/pkg/logs"
	"ft_cni/pkg/network"
	"ft_cni/pkg/utils"

	"github.com/vishvananda/netlink"
)



func DEL(envMap map[string]string, config *load.CNIConfig) error {

	logs.Log.Info("----------------------------")
	logs.Log.Info("----------------------------")
	logs.Log.Info("Begin DEL operation for", "POD", envMap["CNI_CONTAINERID"][:4])

	podIPs, err := utils.GetPodIP(envMap["CNI_NETNS"], envMap["CNI_IFNAME"])

	if err != nil {
		logs.Log.Error("Couldn't Delete & Release POD IPv4.", "ERROR", err)
		return err
	}



	if err != nil {
		logs.Log.Error("Couldn't Load Subnet for IPv4 Deletion.", "ERROR", err)
		return err		
	}

	container_id := envMap["CNI_CONTAINERID"][:4]
	vethHostName := "veth" + container_id

	vethHostEnd, err := netlink.LinkByName(vethHostName)

	if err != nil {
		logs.Log.Error("Couldn't Find Veth Host end.", "ERROR", err)
		return err		
	}

	if err := netlink.LinkDel(vethHostEnd); err != nil {
		logs.Log.Error("Couldn't Delete Veth Host end.", "ERROR", err)
		return err				
	}

	i, err := network.NewIPAM(config.IPAM.Subnet)

	for _, ip := range podIPs {

		logs.Log.Info("Releasing IP", "IP", ip.IP.String())
		if err := i.Release(&ip.IP); err != nil {
			logs.Log.Error("Couldn't release IP Address", "ERROR", err)	
			return err	
		}
	}

	return nil
}