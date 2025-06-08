package targets

import (
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"net"
	"fmt"
	"os"
	"ft_cni/pkg/logs"
	"ft_cni/pkg/network"
	"ft_cni/pkg/load"
	"ft_cni/pkg/result"
)


func CreateVethPair(vethHost, vethPOD string) (*netlink.Veth, error) {

	veth := &netlink.Veth{ // Define veth interfaces (host and pod)
		LinkAttrs: netlink.LinkAttrs{
			Name: vethHost,
		},

		PeerName: vethPOD,
	}

	// Link the interfaces
	if err := netlink.LinkAdd(veth); err != nil {
		logs.Log.Error("Cannot Create Veth Peering between host and Pod.")
		return nil, err
	}

	logs.Log.Info("Veth Peering Created Succefully.")
	return veth, nil
}

func SettingPodEnd(NetnsPath, PodIF, PodVethEndName, PodIP, BridgeIP string) error {

	// Open the cni, safer than os.Open
	podNS, err := netns.GetFromPath(NetnsPath)
	if err != nil {
		logs.Log.Error("Couldn't Open Pod Namespace.")
		return err
	}
	defer podNS.Close()

	// get Link type of Pod Veth end
	veth, err := netlink.LinkByName(PodVethEndName)
	if err != nil {
		logs.Log.Error("Couldn't find POD veth end.")
		return err
	}

	// Move POD veth end to POD namespace
	if err := netlink.LinkSetNsFd(veth, int(podNS)); err != nil {
		logs.Log.Error("Couldn't Move POD veth end to POD namespace.")
		return err
	}
	
	// Save the current namespace
	origNS, err := netns.Get()
	if err != nil {
		logs.Log.Error("Couldn't Save the current namespace.")
		return err
	}
	defer origNS.Close()

	// Switch to the pod's network namespace
	if err := netns.Set(podNS); err != nil {
		logs.Log.Error("Couldn't switch current context to POD namespace.")
		return err
	}

	defer func() {
		netns.Set(origNS)
		logs.Log.Info("Switched back to Host namespace")
	} ()

	if err := netlink.LinkSetName(veth, PodIF); err != nil {
		logs.Log.Error("Couldn't Rename POD veth end.")
		return err
	}

	vethInPod, err := netlink.LinkByName(PodIF) 
	if err != nil {
	    logs.Log.Error("Couldn't find POD veth end.")
	}

	addr, err := netlink.ParseAddr(PodIP)

	if err != nil {
		logs.Log.Error("Couldn't parse Address")
		return err
	}

	if err := netlink.AddrAdd(vethInPod, addr); err != nil {
		logs.Log.Error("Couldn't assign IP to POD interface")
		return err
	}

	if err := netlink.LinkSetUp(vethInPod); err != nil {
		logs.Log.Error("Couldn't set host veth UP.")
		return err;
	}

	gw := net.ParseIP(BridgeIP) // IP of cni0 (bridge on host)

	route := &netlink.Route{
		LinkIndex: vethInPod.Attrs().Index,
		Gw:        gw,
	}

	if err := netlink.RouteAdd(route); err != nil {
		logs.Log.Error("Couldn't Add routing from veth host to bridge.")
		return err;
	}

	logs.Log.Info("Pod Configuration Completed !")

	
	return nil
}

func SettingHostEnd(HostVethEndName string, master netlink.Link) error {

	veth, err := netlink.LinkByName(HostVethEndName)

	if err != nil {
		logs.Log.Error("Couldn't find Host veth end.")
		return err
	}


	if err := netlink.LinkSetMaster(veth, master); err != nil {
		logs.Log.Error("Couldn't Attach host veth end to bridge.")
		return err
	}

	// Set the interface Up of the host
	if err := netlink.LinkSetUp(veth); err != nil {
		logs.Log.Error("Couldn't set host veth UP.")
		return err;
	}

	logs.Log.Info("Host Configuration Completed !")

	return nil

}


func Add(envMap map[string]string, config *load.CNIConfig) {

	logs.Log.Info("Add")

	ipam , err := network.NewIPAM(config.IPAM.Subnet)

	bridge, err := netlink.LinkByName(config.Bridge)
	if err != nil {

		br := &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name: config.Bridge,
			},
		}
		

		if err := netlink.LinkAdd(br); err != nil {
			logs.Log.Error("Couldn't Add Bridge Interface")
		}
		// TODO: Condition if bridge ip exist, prepare it, if not allocate
		br_ip := fmt.Sprintf("%s/%d", config.BridgeIP, 24)
		addr, _ := netlink.ParseAddr(br_ip)

		if err := netlink.AddrAdd(br, addr); err != nil {
			logs.Log.Error("Couldn't assign IP to Bridge interface")
		}

		if err := netlink.LinkSetUp(br); err != nil {
			logs.Log.Error("Couldn't Set Birdge UP")
		}

		bridge = br

	}

	

	container_id := envMap["CNI_CONTAINERID"][:4]
	vethHostName := "veth" + container_id
	
	pod_ip, pod_mask, _ := ipam.Allocate()
	ones, _ := pod_mask.Size()
	cidr := fmt.Sprintf("%s/%d", pod_ip.String(), ones)

	vethPair, err := CreateVethPair(vethHostName, "vethPOD")

	if err != nil {
		os.Exit(1)
	}

	if err := SettingPodEnd(envMap["CNI_NETNS"], envMap["CNI_IFNAME"], vethPair.PeerName, cidr, config.BridgeIP); err != nil {
		netlink.LinkDel(vethPair)
		netlink.LinkDel(bridge)
		ipam.Release(&pod_ip)
		logs.Log.Error("Setting POD end exited with error 1.")
		os.Exit(1)
	}

	if err := SettingHostEnd(vethPair.Name, bridge); err != nil {
		netlink.LinkDel(vethPair)
		netlink.LinkDel(bridge)
		ipam.Release(&pod_ip)
		logs.Log.Error("Setting Host end exited with error 1.")
		os.Exit(1)
	}


	logs.Log.Info("IP of POD %s", cidr)
	logs.Log.Info("IP of bridge %s", config.BridgeIP)

	result.ReturnAdd(envMap["CNI_NETNS"], cidr, config.BridgeIP)
}
