package discovery

import (
	"fmt"
	"ft_cni/pkg/load"
	"ft_cni/pkg/logs"
	"ft_cni/pkg/network"
	"ft_cni/pkg/result"
	"ft_cni/pkg/types"

	"ft_cni/pkg/utils"
	"net"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
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
		logs.Log.Error("Cannot Create Veth Peering between host and Pod.", "ERROR", err)
		return nil, err
	}

	logs.Log.Info("Veth Peering Created Succefully.")
	return veth, nil
}



func SettingPodEnd(
	NetnsPath, 
	PodIF, 
	PodVethEndName,
	bridgeIP,
	PodIP string, 
	Routes []*load.RoutesConfig, 
	CNIResult *types.CNIResult) error {

	// Open the Netns, safer than os.Open
	podNS, err := netns.GetFromPath(NetnsPath)
	if err != nil {
		logs.Log.Error("Couldn't Open Pod Namespace.", "ERROR", err)
		return err
	}
	defer podNS.Close()

	// get Link type of Pod Veth end
	veth, err := netlink.LinkByName(PodVethEndName)
	if err != nil {
		logs.Log.Error("Couldn't find POD veth end in host.", "ERROR", err)
		return err
	}

	// Move POD veth end to POD namespace
	if err := netlink.LinkSetNsFd(veth, int(podNS)); err != nil {
		logs.Log.Error("Couldn't Move POD veth end to POD namespace.", "ERROR", err)
		return err
	}
	
	/*
		Making Golang stick with the current OS Thread to avoid Goscheduler 
		changing Threads and moving my Goroutine to a new thread, which is not  
		set to the POD netns context, as the Setns() only affect it's current thread, 
		it only change the context on the current thread
		which result on returning to host context and attaching IP address 
		to Host's eth0 Interface

		Days spent on this issue = 4

		Whyyyy golang whyyy ???!!!
	*/
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Save the current namespace
	origNS, err := netns.Get()
	if err != nil {
		logs.Log.Error("Couldn't Save the current namespace.", "ERROR", err)
		return err
	}
	defer origNS.Close()

	// Switch to the pod's network namespace
	if err := netns.Set(podNS); err != nil {
		logs.Log.Error("Couldn't switch current context to POD namespace.", "ERROR", err)
		return err
	}

	defer func() {
		netns.Set(origNS)
		logs.Log.Info("Switched back to Host namespace")
	} ()

	if err := netlink.LinkSetName(veth, PodIF); err != nil {
		logs.Log.Error("Couldn't Rename POD veth end.", "ERROR", err)
		return err
	}


	vethInPod, err := netlink.LinkByName(PodIF) 
	if err != nil {
	    logs.Log.Error("Couldn't find POD veth end inside POD.", "ERROR", err)
	}

	addr, err := netlink.ParseAddr(PodIP)

	if err != nil {
		logs.Log.Error("Couldn't parse Address", "ERROR", err)
		return err
	}

	if err := netlink.AddrAdd(vethInPod, addr); err != nil {
		logs.Log.Error("Couldn't assign IP to POD interface", "ERROR", err)
		return err
	}

	if err := netlink.LinkSetUp(vethInPod); err != nil {
		logs.Log.Error("Couldn't set host veth UP.", "ERROR", err)
		return err;
	}

	// Result gatehring
	CNIResult.Interfaces = append(CNIResult.Interfaces, types.Interface{
		Name: PodIF,
		MAC: vethInPod.Attrs().HardwareAddr.String(),
		Sandbox: NetnsPath,
	})


	//------------------- Routing ------------------------

	for _, routes := range Routes {

		gw := net.ParseIP(bridgeIP)
		
		if routes.Gw != "" {
			gw = net.ParseIP(routes.Gw) 
		}
		
		_, dst, _ := net.ParseCIDR(routes.Dst)

		route := &netlink.Route{
			LinkIndex: vethInPod.Attrs().Index,
			Gw:  gw,
			Dst: dst,
		}

		if err := netlink.RouteAdd(route); err != nil {
			logs.Log.Error("Couldn't Add routing from veth host to bridge.", "ERROR", err)
			return err;
		}

		CNIResult.Routes = append(CNIResult.Routes, types.Route{
			Dst: routes.Dst,
			GW: routes.Gw,
		})
	}


	logs.Log.Info("Pod Configuration Completed !")


	return nil	
}

func SettingHostEnd(HostVethEndName string, master netlink.Link, CNIResult *types.CNIResult) error {

	veth, err := netlink.LinkByName(HostVethEndName)

	if err != nil {
		logs.Log.Error("Couldn't find Host veth end.", "ERROR", err)
		return err
	}

	if err := netlink.LinkSetMaster(veth, master); err != nil {
		logs.Log.Error("Couldn't Attach host veth end to bridge.", "ERROR", err)
		return err
	}

	// Set the interface Up of the host
	if err := netlink.LinkSetUp(veth); err != nil {
		logs.Log.Error("Couldn't set host veth UP.", "ERROR", err)
		return err
	}

	logs.Log.Info("Host Configuration Completed !")

	CNIResult.Interfaces = append(CNIResult.Interfaces, types.Interface{
		Name: HostVethEndName,
		MAC: veth.Attrs().HardwareAddr.String(),
	})

	return nil

}

func LoadDNS(DNS *load.DNSConfig, CNIResult *types.CNIResult) {

	if DNS == nil {
		return 
	}

	CNIResult.DNS = &types.DNSConfig{
		Nameservers: make([]string, 0),
		Options: make([]string, 0),
		Search: make([]string, 0),
		Domain: "",
	}

	if DNS.Nameservers != nil {

		for _, ns := range DNS.Nameservers {
			CNIResult.DNS.Nameservers = append(CNIResult.DNS.Nameservers, ns)
		}

	}
	
	if DNS.Options != nil {

		for _, op := range DNS.Options {
			CNIResult.DNS.Options = append(CNIResult.DNS.Options, op)
		}
		
	}
	
	if DNS.Search != nil {

		for _, sc := range DNS.Search {
			CNIResult.DNS.Search = append(CNIResult.DNS.Search, sc)
		}

	}
	
	if DNS.Domain != "" {

		CNIResult.DNS.Domain = DNS.Domain

	} 

}


func Add(envMap map[string]string, config *load.CNIConfig) error {

    logs.Log.Info("----------------------------")
	logs.Log.Info("----------------------------")
	logs.Log.Info("Begin ADD operation for", "POD", envMap["CNI_CONTAINERID"][:4])

	//---------------- Create Result Object + Initialization -------------

	CNIResult := &types.CNIResult{
	    CNIVersion: envMap["CNI_VERSION"],
	    Interfaces: make([]types.Interface, 0), 
	    IPs:        make([]types.IPConfig, 0),
	    Routes:     make([]types.Route, 0),   
	}



	// --------------- Create new IPAM Subnet ------------
    ipam, err := network.NewIPAM(config.IPAM.Subnet)
	if err != nil {
		logs.Log.Error("Could create new IPAM subnet", config.IPAM.Subnet, "ERROR", err)
		return err
	}


	// --------------- Parse IPAM Subnet CIDR ------------
	_, ip_net, err := net.ParseCIDR(config.IPAM.Subnet) 
	if err != nil {
		logs.Log.Error("Could create parse subnet CIDR", config.IPAM.Subnet, "ERROR", err)
		return err		
	}
    
    // ----------- Attempt to find the bridge --------------
	logs.Log.Info("Searching for Bridge", "BRIDGE", config.Bridge)
    bridge, err := netlink.LinkByName(config.Bridge)
    var brIP net.IP

    if err != nil { // --------- Bridge does NOT exist, create it ----------------
        logs.Log.Info("Bridge not found. Creating it......", "BRIDGE", config.Bridge)
        br := &netlink.Bridge{
            LinkAttrs: netlink.LinkAttrs{
                Name: config.Bridge,
            },
        }

        if err := netlink.LinkAdd(br); err != nil {
            logs.Log.Error("Couldn't Add Bridge Interface", "ERROR", err)
            return err
        }
		logs.Log.Info("Bridge created succefully", "BRIDGE", config.Bridge)
        // ------------- IP Allocation for NEW bridge ----------
        var brCIDR string
		logs.Log.Info("Attaching IP Address for", "BRIDGE", config.Bridge)
        if config.BridgeIP != "" {
			logs.Log.Info("Found Bridge IP Address at Conflist", "BRIDGE", config.Bridge)
            ones, _ := ip_net.Mask.Size()
            brCIDR = fmt.Sprintf("%s/%d", config.BridgeIP, ones)
            allocatedIP, _, err := ipam.Allocate(config.BridgeIP)
            if err != nil {
                logs.Log.Error("Couldn't Allocate specified IP for Bridge Interface", "ERROR", err)
				logs.Log.Info("Cleaning... Deleting Bridge", "BRIDGE", config.Bridge)
                if err := netlink.LinkDel(br); err != nil {
					logs.Log.Error("Couldn't delete Bridge", "ERROR", err)
				}
                return err
            }
            brIP = allocatedIP
			logs.Log.Info("IP Address is allocated for", "BRIDGE", config.Bridge)
        } else {
			logs.Log.Info("No IP Address was found in Conflist for", "BRIDGE", config.Bridge)
			logs.Log.Info("Allocating new IP Address from IPAM", "SUBNET", config.IPAM.Subnet)

            allocatedIP, br_mask, err := ipam.Allocate("")
            if err != nil {
                logs.Log.Error("Couldn't Allocate specified IP for Bridge Interface", "ERROR", err)
				logs.Log.Info("Cleaning... Deleting Bridge", "BRIDGE", config.Bridge)
                if err := netlink.LinkDel(br); err != nil {
					logs.Log.Error("Couldn't delete Bridge", "ERROR", err)
				}
                return err
            }
            ones, _ := br_mask.Size()
            brCIDR = fmt.Sprintf("%s/%d", allocatedIP.String(), ones)
			logs.Log.Info("IP Address is allocated for", "BRIDGE", config.Bridge)
            brIP = allocatedIP
        }

        addr, err := netlink.ParseAddr(brCIDR)
		if err != nil {
			logs.Log.Error("Couldn't Parse Bridge IP Address", "ERROR", err)
		}
        if err := netlink.AddrAdd(br, addr); err != nil {
            logs.Log.Error("Couldn't assign IP to Bridge interface", "ERROR", err)
			logs.Log.Info("Cleaning.......Deleting Bridge Interafce and releasing IP address")
            if err := netlink.LinkDel(br); err != nil {
				logs.Log.Error("Couldn't delete Bridge", "ERROR", err)
			}
            if err := ipam.Release(&brIP); err != nil {
				logs.Log.Error("Couldn't Release IP Address", "ERROR", err)
			}
            return err
        }

        if err := netlink.LinkSetUp(br); err != nil {
            logs.Log.Error("Couldn't set Bridge UP", "ERROR", err)
			logs.Log.Info("Cleaning.......Deleting Bridge Interafce and releasing IP address")
            if err := netlink.LinkDel(br); err != nil {
				logs.Log.Error("Couldn't delete Bridge", "ERROR", err)
			}
            if err := ipam.Release(&brIP); err != nil {
				logs.Log.Error("Couldn't Release IP Address", "ERROR", err)
			}
            return err
        }
        //---------- Update bridge variable with the new link -------------------
        bridge = br 

    } else { // Bridge ALREADY exists
        logs.Log.Info("Bridge already exists. Getting its configuration.", "BRIDGE", config.Bridge)

        addrs, err := netlink.AddrList(bridge, netlink.FAMILY_V4)
        if err != nil || len(addrs) == 0 {
            logs.Log.Error("Couldn't get IP for existing bridge", "ERROR", err)
            return err
        }
		//------------ Get the Primary IPv4 address -----------------
        brIP = addrs[0].IP 
    }

	
	//--------------- Setting name for Host Veth end -----------------
	container_id := envMap["CNI_CONTAINERID"][:4]
	vethHostName := "veth" + container_id
	vethPODName := "pod" + container_id
	
	//--------------- Create VethPair -----------------
	logs.Log.Info("Creating Veth pair for POD and Host", "VethHost", vethHostName)
	vethPair, err := CreateVethPair(vethHostName, vethPODName)

	if err != nil {
		return err
	}
    defer func() {
		if err != nil {
			netlink.LinkDel(vethPair)
		}
	} ()


	//--------------- Allocate IP Address for POD -----------------
	logs.Log.Info("Allocating new IP Address for", "POD", envMap["CNI_CONTAINERID"][:4])
	pod_ip, pod_mask, err := ipam.Allocate("")

	if err != nil {
		logs.Log.Error("Couldn't Allocate IP for POD.", "ERROR", err)
		return err
	}
	
	// Getting Subnet mask as String
	ones, _ := pod_mask.Size()
	cidr := fmt.Sprintf("%s/%d", pod_ip.String(), ones)


	//------------------- Setting POD End ----------------------
	if err := SettingPodEnd(
		envMap["CNI_NETNS"], 
		envMap["CNI_IFNAME"], 
		vethPair.PeerName,
		brIP.String(),
		cidr, 
		config.IPAM.Routes, 
		CNIResult,
	); err != nil {
		if err := netlink.LinkDel(vethPair); err != nil {
			logs.Log.Error("Couldn't delete vethPair", "ERROR", err)
		}
		if err := ipam.Release(&pod_ip); err != nil {
			logs.Log.Error("Couldn't release IP Address for POD", "ERROR", err)
		}
		logs.Log.Error("Setting POD end exited with error 1.")
		return err
	}

	if err := SettingHostEnd(vethPair.Name, bridge, CNIResult); err != nil {
		if err := netlink.LinkDel(vethPair); err != nil {
			logs.Log.Error("Couldn't delete vethPair", "ERROR", err)
		}
		if err := ipam.Release(&pod_ip); err != nil {
			logs.Log.Error("Couldn't release IP Address for POD", "ERROR", err)
		}
		logs.Log.Error("Setting Host end exited with error 1.")
		return err
	}

	//-------------- Get POD IP Addresses -------------------
	Addrs, err := utils.GetPodIP(envMap["CNI_NETNS"], envMap["CNI_IFNAME"])

	for i, Address := range Addrs {

		ipConf := types.IPConfig{
        	Address: Address.IPNet.String(),
        	Interface: i, 
    	}
		ipConf.Version = "4"

		CNIResult.IPs = append(CNIResult.IPs, ipConf)
	}

	LoadDNS(config.IPAM.DNS, CNIResult)

	return result.ReturnAdd(CNIResult)
}
