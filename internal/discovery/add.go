package discovery

import (
	"fmt"
	"net"
	"rbati-cni/internal/load"
	"rbati-cni/internal/logs"
	"rbati-cni/internal/result"
	"rbati-cni/internal/types"
	"rbati-cni/internal/utils"
	"rbati-cni/internal/ipam"
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
	}()

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
		return err
	}

	// Result gatehring
	CNIResult.Interfaces = append(CNIResult.Interfaces, types.Interface{
		Name:    PodIF,
		MAC:     vethInPod.Attrs().HardwareAddr.String(),
		Sandbox: NetnsPath,
	})

	//------------------- Routing ------------------------

	if len(Routes) == 0 {
		logs.Log.Info("No routes specified in config, adding default route to", "GW", bridgeIP)

		gw := net.ParseIP(bridgeIP)
		_, dst, _ := net.ParseCIDR("0.0.0.0/0")

		route := &netlink.Route{
			LinkIndex: vethInPod.Attrs().Index,
			Gw:        gw,
			Dst:       dst,
		}

		if err := netlink.RouteAdd(route); err != nil {
			logs.Log.Error("Couldn't Add fallback default route", "ERROR", err)
			return err
		}
		
		CNIResult.Routes = append(CNIResult.Routes, types.Route{
			Dst: dst.String(),
			GW:  gw.String(),
		})
	}

	for _, routes := range Routes {

		gw := net.ParseIP(bridgeIP)

		if routes.Gw != "" { // in each route, if gw is empty, then gw is the bridge
			gw = net.ParseIP(routes.Gw)
		}

		_, dst, _ := net.ParseCIDR("0.0.0.0/0") // Default route

		if routes.Dst != "" {
			_, dst, _ = net.ParseCIDR(routes.Dst)
		}
		
		route := &netlink.Route{
			LinkIndex: vethInPod.Attrs().Index,
			Gw:        gw,
			Dst:       dst,
		}

		if err := netlink.RouteAdd(route); err != nil {
			logs.Log.Error("Couldn't Add routing from veth host to bridge.", "ERROR", err)
			return err
		}

		CNIResult.Routes = append(CNIResult.Routes, types.Route{
			Dst: dst.String(),
			GW:  gw.String(),
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
	
	if err := netlink.LinkSetHairpin(veth, true); err != nil {
		logs.Log.Error("Couldn't set Hairpin mode !", "ERROR", err)
		return err
	}

	logs.Log.Info("Set Harpin mode.")
	logs.Log.Info("Host Configuration Completed !")

	CNIResult.Interfaces = append(CNIResult.Interfaces, types.Interface{
		Name: HostVethEndName,
		MAC:  veth.Attrs().HardwareAddr.String(),
	})

	return nil

}

func LoadDNS(DNS *load.DNSConfig, CNIResult *types.CNIResult) {

	if DNS == nil {
		return
	}

	CNIResult.DNS = &types.DNSConfig{
		Nameservers: make([]string, 0),
		Options:     make([]string, 0),
		Search:      make([]string, 0),
		Domain:      "",
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

func Add(envMap map[string]string, config *load.CNIConfig, bridgeName string, ipam *ipam.IPAM) error {

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


	// --------------- Get Bridge ------------

	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		logs.Log.Error("Couldn't find bridge", "ERROR", err)
	}

	addrs, err := netlink.AddrList(bridge, netlink.FAMILY_V4)
	if err != nil || len(addrs) == 0 {
		logs.Log.Error("Couldn't get IP for existing bridge", "ERROR", err)
		return err
	}

	//------------ Get the Primary IPv4 address -----------------
	brIP := addrs[0].IP

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
	}()

	//--------------- Allocate IP Address for POD -----------------
	logs.Log.Info("Allocating new IP Address for", "POD", envMap["CNI_CONTAINERID"][:4])

	podIP, podMask, err := ipam.Allocate("")

	if err != nil {
		logs.Log.Error("Couldn't Allocate IP for POD.", "ERROR", err)
		return err
	}

	ones, _ := podMask.Size()

	cidr := fmt.Sprintf("%s/%d", podIP, ones)

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

		if err := ipam.Release(podIP.String()); err != nil {
			logs.Log.Error("Couldn't release IP Address for POD", "ERROR", err)
		}
		logs.Log.Error("Released POD IP", "IP", podIP.String())
		logs.Log.Error("Setting POD end exited with error 1.")
		return err
	}

	if err := SettingHostEnd(vethPair.Name, bridge, CNIResult); err != nil {
		if err := netlink.LinkDel(vethPair); err != nil {
			logs.Log.Error("Couldn't delete vethPair", "ERROR", err)
		}
		if err := ipam.Release(podIP.String()); err != nil {
			logs.Log.Error("Couldn't release IP Address for POD", "ERROR", err)
		}
		logs.Log.Error("Released POD IP", "IP", podIP.String())
		logs.Log.Error("Setting Host end exited with error 1.")
		return err
	}

	//-------------- Get POD IP Addresses -------------------
	Addrs, err := utils.GetPodIP(envMap["CNI_NETNS"], envMap["CNI_IFNAME"])

	for i, Address := range Addrs {

		ipConf := types.IPConfig{
			Address:   Address.IPNet.String(),
			Interface: i,
		}
		ipConf.Version = "4"

		CNIResult.IPs = append(CNIResult.IPs, ipConf)
	}

	LoadDNS(config.IPAM.DNS, CNIResult)

	return result.ReturnAdd(CNIResult)
}