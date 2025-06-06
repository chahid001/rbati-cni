package targets

import (
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"net"
	"fmt"
	"os"
	"encoding/json"
	"ft_cni/pkg/logger"
)


func CreateVethPair(vethHost, vethPOD string) error {

	veth := &netlink.Veth{ //Define veth interfaces (host and pod)
		LinkAttrs: netlink.LinkAttrs{
			Name: vethHost,
		},

		PeerName: vethPOD,
	}

	// Link the interfaces
	if err := netlink.LinkAdd(veth); err != nil {
		return err;
	}
	
	logger.Log.Println("Veth Created")
	return nil
}

func SettingPodNet(NetnsPath, vethPOD string) error {

	/*
		First step 
		Get the host Default Namespace for later use
	*/

	hostNS, err := netns.Get()
	
	if err != nil {
		return err
	}

	defer hostNS.Close()

	// -------------------------------------
	/*
		Second Step
		Getting the pod Namespace and setting it for an active fd
		LinkSetNsFd need an active ns fd	
	*/

	// Open the cni, safer than os.Open
	podNS, err := netns.GetFromPath(NetnsPath)

	if err != nil {
		return err
	}
	defer podNS.Close()
	// get Link type of veth of pod
	veth, err := netlink.LinkByName(vethPOD)

	if err != nil {
		return err
	}

	// set the veth of the pod to the pod NS
	if err := netlink.LinkSetNsFd(veth, int(podNS)); err != nil {
		return err
	}
	
	// Save the current namespace
	origNS, err := netns.Get()
	if err != nil {
		return err
	}
	defer origNS.Close()

	// Switch to the pod's network namespace
	if err := netns.Set(podNS); err != nil {
		return err
	}
	defer netns.Set(origNS)


	vethInNS, err := netlink.LinkByName(vethPOD)
	if err != nil {
		return err
	}

	if err := netlink.LinkSetName(vethInNS, "eth0"); err != nil {
		return err
	}

	logger.Log.Println("Pod initialized")
	return nil
}

func SettingHostNet(vethHost string, master netlink.Link) error {

	veth, err := netlink.LinkByName(vethHost)

	if err != nil {
		return err
	}


	if err := netlink.LinkSetMaster(veth, master); err != nil {
		return err
	}

	// Set the interface Up of the host
	if err := netlink.LinkSetUp(veth); err != nil {
		return err;
	}

	logger.Log.Println("Host initialized")

	return nil

}

func SettingIpam(NetnsPath string) error {

	hostNS, err := netns.Get()
	
	if err != nil {
		return err
	}

	defer hostNS.Close()

	// -------------------------------------
	/*
		Second Step
		Getting the pod Namespace and setting it for an active fd
		LinkSetNsFd need an active ns fd	
	*/

	// Open the cni, safer than os.Open
	podNS, err := netns.GetFromPath(NetnsPath)

	if err != nil {
		return err
	}
	defer podNS.Close()
		// Save the current namespace
	origNS, err := netns.Get()
	if err != nil {
		return err
	}
	defer origNS.Close()
	if err := netns.Set(podNS); err != nil {
		return err
	}
	defer netns.Set(origNS)
	// get Link type of veth of pod
	veth, err := netlink.LinkByName("eth0")

	if err != nil {
		return err
	}




	addr, _ := netlink.ParseAddr("10.42.0.12/24")



	netlink.AddrAdd(veth, addr)
	netlink.LinkSetUp(veth)

	gw := net.ParseIP("10.42.0.1") // IP of cni0 (bridge on host)

	route := &netlink.Route{
		LinkIndex: veth.Attrs().Index,
		Gw:        gw,
	}

	netlink.RouteAdd(route)


	return nil
}


func Add(envMap map[string]string) {

	err := logger.Init("/var/log/mycni.log")
	if err != nil {
		fmt.Println("Failed to initialize logger:", err)
		os.Exit(1)
	}

	logger.Log.Println("Logger initialized")


	bridge, err := netlink.LinkByName("cni-br0")

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
		if err := netlink.AddrAdd(br, addr); err != nil{
		    logger.Log.Println("sgs")
		}

		if err := netlink.LinkSetUp(br); err != nil {
			logger.Log.Println("sgs")
		}

		bridge = br

	}

	container_id := envMap["CNI_CONTAINERID"][:4]

	vethHostName := "veth" + container_id


	CreateVethPair("vethPOD", vethHostName)
	SettingPodNet(envMap["CNI_NETNS"], "vethPOD")
	SettingHostNet(vethHostName, bridge)
	SettingIpam(envMap["CNI_NETNS"])

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
				"address": "10.42.0.12/24",
				"gateway": "10.42.0.1",
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