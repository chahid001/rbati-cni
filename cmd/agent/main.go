package main

import (
	"log"
	"os"
	"path/filepath"
	"rbati-cni/internal/controller"
	"rbati-cni/internal/ipam"
	"rbati-cni/internal/kube"
	"rbati-cni/internal/load"
	"rbati-cni/internal/metrics"
	"rbati-cni/internal/network"
	"rbati-cni/internal/types"
	"rbati-cni/internal/utils"
	"strings"
	"time"

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

	if err := network.EnableNat(podSubnet, types.VXLAN_NAME); err != nil {
		log.Printf("[ERROR] Failed to Enable nat: %v\n", err)
		os.Exit(1)
	}

	//--------------- Setup VXLAN ------------------
    const maxRetries = 10
    const retryDelay = 3 * time.Second

    for i := 0; i < maxRetries; i++ {

        err = network.SetupVXLAN(types.VXLAN_NAME, configJSON.IFace, bridge, 1001)
        if err == nil {
            break 
        }

        if strings.Contains(err.Error(), "cannot assign requested address") {
            log.Printf("[WARN] VXLAN not fully ready (%v). Retrying in %v...", err, retryDelay)
            time.Sleep(retryDelay)
            continue
        } else {
            log.Fatalf("[FATAL] Unrecoverable error during VXLAN setup: %v", err)
        }
    }
    if err != nil {
        log.Fatalf("[FATAL] Failed to create VXLAN device after %d retries: %v", maxRetries, err)
    }

	kube.PatchNode(clientset, kube.GetNodeName(), utils.GetMAC(types.VXLAN_NAME))

	log.Printf("[Agent] Finished the setup with success !")


	//--------------- Init Metrics ------------------

	ipam_path := filepath.Join(types.IPAM_FODLER_PATH, types.IPAM_DB_FILENAME)
	ipamInstance, err := ipam.NewIPAM(podSubnet, ipam_path)

    if err != nil {
        log.Fatalf("[FATAL] Failed to init IPAM for metrics: %v", err)
    }

	metrics.Init(ipamInstance)
	 log.Printf("[INFO] Metrics endpoint started on :9090")

	//--------------- Start Informers ------------------
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