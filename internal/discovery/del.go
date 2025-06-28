package discovery

import (
	"context"
	"rbati-cni/internal/load"
	"rbati-cni/internal/logs"
	"rbati-cni/internal/utils"

	pb "rbati-cni/api"

	"github.com/vishvananda/netlink"
)

func DEL(envMap map[string]string, config *load.CNIConfig, client pb.IPAMServiceClient) error {

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

	for _, ip := range podIPs {

		logs.Log.Info("Releasing IP", "IP", ip.IP.String())
		_, err := client.ReleaseIP(context.Background(), &pb.ReleaseRequest{
			Ip: ip.IP.String(),
		})
		if err != nil {
			logs.Log.Error("Couldn't release IP Address for POD", "ERROR", err)
			return err
		}
	}

	return nil
}
