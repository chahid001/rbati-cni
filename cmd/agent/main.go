package main

import (
	"log"
	"os"
	"rbati-cni/internal/controller"
	"rbati-cni/internal/kube"
	"rbati-cni/internal/load"
	"rbati-cni/internal/network"
	"rbati-cni/internal/types"
	"rbati-cni/internal/utils"

	"github.com/vishvananda/netlink"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var configJSON *load.CNIConfig


func init() {

	configJSON = load.LoadFromFile(types.CONFIG_FILE_PATH)

}



func main() {

	log.Printf("[INFO] [Agent 1/2] Agent Setup starting on node: %s\n", kube.GetNodeName())

	bridge := types.BRIDGE_NAME

	if configJSON.Bridge != "" {
		bridge = configJSON.Bridge
	}

	// ---------------- Kubernetes API --------------------
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("[ERROR] Failed to get in-cluster config: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("[ERROR] Failed to create clientset: %v\n", err)
		os.Exit(1)
	}

	//--------------- Get PodCIDR ------------------
	podSubnet, err := kube.GetNodePodCIDR(clientset, kube.GetNodeName())
	if err != nil {
		log.Printf("[ERROR] Failed to get pod Subnet: %v\n", err)
		os.Exit(1)
	}

	//----- Pass NodeName as ProducerConsumer pattern -------------
	if err := utils.ProducerConsumer(podSubnet); err != nil {
		log.Printf("[ERROR] %v\n", err)
		os.Exit(1)		
	}

	//--------------- Get Gateway IP ------------------
	gatewayIP, err := network.GetGatewayIP(podSubnet)
	if err != nil {
		log.Printf("[ERROR] Failed to get Gateway IP: %v\n", err)
		os.Exit(1)		
	}	

	//--------------- Setup Bridge ------------------
	if err := network.SetupBridge(bridge, gatewayIP); err != nil {
		log.Printf("[ERROR] Failed to create Bridge device: %v\n", err)
		os.Exit(1)
	}

	//--------------- Enable NAT ------------------

	if err := network.EnableNat(podSubnet, configJSON.IFace); err != nil {
		log.Printf("[ERROR] Failed to Enable nat: %v\n", err)
		os.Exit(1)
	}

	//--------------- Setup VXLAN ------------------
	if err := network.SetupVXLAN(types.VXLAN_NAME, configJSON.IFace, bridge, 1001); err != nil {
		log.Printf("[ERROR] Failed to create VXLAN device: %v\n", err)
		os.Exit(1)
	}

	kube.PatchNode(clientset, kube.GetNodeName(), utils.GetMAC(types.VXLAN_NAME))

	log.Printf("[Agent] Finished the setup with success !")
	log.Printf("[INFO] [Controller 2/2] Controller starting on node: %s\n", kube.GetNodeName())

	vxlan, err := netlink.LinkByName(types.VXLAN_NAME)
	if err != nil {
	    log.Fatalf("[FATAL] Failed to find VXLAN link %s after setup: %v", types.VXLAN_NAME, err)
		os.Exit(1)
	}


	Controller := controller.NewController(clientset, kube.GetNodeName(), vxlan)

	stopCh := make(chan struct{})
	defer close(stopCh)

	Controller.Run(stopCh)

}