package targets

import (
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"net"
	"fmt"
	"os"
	"encoding/json"
	"ft_cni/pkg/logger"
	"ft_cni/pkg/network"
)


func CreateVethPair(vethHost, vethPOD string) error {

	veth := &netlink.Veth{ // Define veth interfaces (host and pod)
		LinkAttrs: netlink.LinkAttrs{
			Name: vethHost,
		},

		PeerName: vethPOD,
	}

	// Link the interfaces
	if err := netlink.LinkAdd(veth); err != nil {
		logger.Log.Println("Cannot Create Veth Peering between host and Pod.")
		return err
	}
	
	logger.Log.Println("Veth Peering Created Succefully.")
	return nil
}

func SettingPodNet(NetnsPath, vethPOD string) error {

	hostNS, err := netns.Get()
	
	if err != nil {
		logger.Log.Println("Failed to get Host namespace.")
		return err
	}

	defer hostNS.Close()


	// Open the cni, safer than os.Open
	podNS, err := netns.GetFromPath(NetnsPath)
	if err != nil {
		logger.Log.Println("Couldn't Open Pod Namespace.")
		return err
	}
	defer podNS.Close()

	// get Link type of Pod Veth end
	veth, err := netlink.LinkByName(vethPOD)

	if err != nil {
		logger.Log.Println("Couldn't find POD veth end.")
		return err
	}

	// Move POD veth end to POD namespace
	if err := netlink.LinkSetNsFd(veth, int(podNS)); err != nil {
		logger.Log.Println("Couldn't Move POD veth end to POD namespace.")
		return err
	}
	
	// Save the current namespace
	origNS, err := netns.Get()
	if err != nil {
		logger.Log.Println("Couldn't Save the current namespace.")
		return err
	}
	defer origNS.Close()

	// Switch to the pod's network namespace
	if err := netns.Set(podNS); err != nil {
		logger.Log.Println("Couldn't switch current context to POD namespace.")
		return err
	}

	defer func() {
		netns.Set(origNS)
		logger.Log.Println("Switched back to Host namespace")
	} ()

	if err := netlink.LinkSetName(veth, "eth0"); err != nil {
		logger.Log.Println("Couldn't Rename POD veth end.")
		return err
	}

	logger.Log.Println("Pod Configuration Completed !")

	
	return nil
}

func SettingHostNet(vethHost string, master netlink.Link) error {

	veth, err := netlink.LinkByName(vethHost)

	if err != nil {
		logger.Log.Println("Couldn't find Host veth end.")
		return err
	}


	if err := netlink.LinkSetMaster(veth, master); err != nil {
		logger.Log.Println("Couldn't Attach host veth end to bridge.")
		return err
	}

	// Set the interface Up of the host
	if err := netlink.LinkSetUp(veth); err != nil {
		logger.Log.Println("Couldn't set host veth UP.")
		return err;
	}

	logger.Log.Println("Host Configuration Completed !")

	return nil

}

func SettingIpam(NetnsPath, Address, BridgeIP string) error {

	podNS, err := netns.GetFromPath(NetnsPath)
	if err != nil {
		logger.Log.Println("Couldn't Open Pod Namespace.")
		return err
	}
	defer podNS.Close()


	origNS, err := netns.Get()
	if err != nil {
		logger.Log.Println("Couldn't Save the current namespace.")
		return err
	}
	defer origNS.Close()

	if err := netns.Set(podNS); err != nil {
		logger.Log.Println("Couldn't switch current context to POD namespace.")
		return err
	}

	defer func() {
		netns.Set(origNS)
		logger.Log.Println("Switched back to Host namespace")
	} ()

	veth, err := netlink.LinkByName("eth0")
	if err != nil {
		logger.Log.Println("Couldn't find Pod veth end.")
		return err
	}

	addr, _ := netlink.ParseAddr(Address)

	if err := netlink.AddrAdd(veth, addr); err != nil {
		logger.Log.Println("Couldn't assign IP to POD interface")
		return err
	}

	if err := netlink.LinkSetUp(veth); err != nil {
		logger.Log.Println("Couldn't set host veth UP.")
		return err;
	}

	gw := net.ParseIP(BridgeIP) // IP of cni0 (bridge on host)

	route := &netlink.Route{
		LinkIndex: veth.Attrs().Index,
		Gw:        gw,
	}

	if err := netlink.RouteAdd(route); err != nil {
		logger.Log.Println("Couldn't Add routing from veth host to bridge.")
		return err;
	}


	return nil
}


func Add(envMap map[string]string) {

	err := logger.Init("/var/log/mycni.log")
	if err != nil {
		fmt.Println("Failed to initialize logger:", err)
		os.Exit(1)
	}

	logger.Log.Println("Logger initialized")


	ipam , err := network.NewIPAM("10.42.0.0/24")

	bridge, err := netlink.LinkByName("cni_br0")
	br_ip := "10.42.0.1"
	if err != nil {

		br := &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name: "cni_br0",
			},
		}


		if err := netlink.LinkAdd(br); err != nil {
			logger.Log.Println("sgs")
		}

		addr, _ := netlink.ParseAddr("10.42.0.1/24")

		if err := netlink.AddrAdd(br, addr); err != nil {
			logger.Log.Println("Couldn't assign IP to Bridge interface")
		}

		if err := netlink.LinkSetUp(br); err != nil {
			logger.Log.Println("sgs")
		}


		bridge = br

	}

	

	container_id := envMap["CNI_CONTAINERID"][:4]

	vethHostName := "veth" + container_id

	pod_ip, pod_mask, _ := ipam.Allocate()
	

	CreateVethPair("vethPOD", vethHostName)
	SettingPodNet(envMap["CNI_NETNS"], "vethPOD")
	SettingHostNet(vethHostName, bridge)

	ones, _ := pod_mask.Size()
	cidr := fmt.Sprintf("%s/%d", pod_ip.String(), ones)
	logger.Log.Println("IP of POD %s", cidr)
	logger.Log.Println("IP of bridge %s", br_ip)
	SettingIpam(envMap["CNI_NETNS"], cidr, br_ip)

	result := map[string]interface{}{
		"cniVersion": "0.4.0",
		"interfaces": []map[string]string{
			{
				"name":    "eth0",
				"sandbox": envMap["CNI_NETNS"], 
			},
		},
		"ips": []map[string]string{
			{
				"version": "4",
				"address": cidr,
				"gateway": br_ip,
			},
		},
		"dns": map[string]string{},
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}