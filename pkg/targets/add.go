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

    ipam, _ := network.NewIPAM(config.IPAM.Subnet)
    _, ip_net, _ := net.ParseCIDR(config.IPAM.Subnet)
    
    // Attempt to find the bridge
    bridge, err := netlink.LinkByName(config.Bridge)
    var brIP net.IP

    if err != nil { // Bridge does NOT exist, create it
        logs.Log.Info("Bridge %s not found. Creating it.", config.Bridge)
        br := &netlink.Bridge{
            LinkAttrs: netlink.LinkAttrs{
                Name: config.Bridge,
            },
        }

        if err := netlink.LinkAdd(br); err != nil {
            logs.Log.Error("Couldn't Add Bridge Interface")
            os.Exit(1)
        }

        // --- IP Allocation for NEW bridge ---
        var brCIDR string
        if config.BridgeIP != "" {
            ones, _ := ip_net.Mask.Size()
            brCIDR = fmt.Sprintf("%s/%d", config.BridgeIP, ones)
            allocatedIP, _, err := ipam.Allocate(config.BridgeIP)
			logs.Log.Error("error: ", err)
            if err != nil {
                logs.Log.Error("Couldn't Allocate specified IP for Bridge Interface")
                netlink.LinkDel(br)
                os.Exit(1)
            }
            brIP = allocatedIP
        } else {
            allocatedIP, br_mask, err := ipam.Allocate("")
            if err != nil {
                logs.Log.Error("Couldn't Allocate new IP for Bridge Interface")
                netlink.LinkDel(br)
                os.Exit(1)
            }
            ones, _ := br_mask.Size()
            brCIDR = fmt.Sprintf("%s/%d", allocatedIP.String(), ones)
            brIP = allocatedIP
        }

        addr, _ := netlink.ParseAddr(brCIDR)
        if err := netlink.AddrAdd(br, addr); err != nil {
            logs.Log.Error("Couldn't assign IP to Bridge interface")
            netlink.LinkDel(br)
            ipam.Release(&brIP)
            os.Exit(1)
        }

        if err := netlink.LinkSetUp(br); err != nil {
            logs.Log.Error("Couldn't Set Bridge UP")
            netlink.LinkDel(br)
            ipam.Release(&brIP)
            os.Exit(1)
        }
        
        bridge = br // Update bridge variable with the new link

    } else { // Bridge ALREADY exists
        logs.Log.Info("Bridge %s already exists. Getting its configuration.", config.Bridge)

        addrs, err := netlink.AddrList(bridge, netlink.FAMILY_V4)
        if err != nil || len(addrs) == 0 {
            logs.Log.Error("Couldn't get IP for existing bridge: %v", err)
            os.Exit(1)
        }
        brIP = addrs[0].IP // Get the first IPv4 address
    }

	

	container_id := envMap["CNI_CONTAINERID"][:4]
	vethHostName := "veth" + container_id
	
	pod_ip, pod_mask, err := ipam.Allocate("")
	if err != nil {
		logs.Log.Error("Couldn't Allocate IP for POD.")
		os.Exit(1)
	}
	logs.Log.Error(pod_ip.String())
	ones, _ := pod_mask.Size()
	cidr := fmt.Sprintf("%s/%d", pod_ip.String(), ones)

	vethPair, err := CreateVethPair(vethHostName, "vethPOD")

	if err != nil {
		os.Exit(1)
	}

	logs.Log.Error("dfdsf", brIP.String())
	if err := SettingPodEnd(envMap["CNI_NETNS"], envMap["CNI_IFNAME"], vethPair.PeerName, cidr, brIP.String()); err != nil {
		netlink.LinkDel(vethPair)
		ipam.Release(&pod_ip)
		logs.Log.Error("Setting POD end exited with error 1.")
		os.Exit(1)
	}

	if err := SettingHostEnd(vethPair.Name, bridge); err != nil {
		netlink.LinkDel(vethPair)
		ipam.Release(&pod_ip)
		logs.Log.Error("Setting Host end exited with error 1.")
		os.Exit(1)
	}


	logs.Log.Info("IP of POD %s", cidr)
	logs.Log.Info("IP of bridge %s", config.BridgeIP)

	result.ReturnAdd(envMap["CNI_NETNS"], cidr,  brIP.String())
}
