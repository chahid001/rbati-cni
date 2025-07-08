package main

import (
	"log"
	"os"
	"rbati-cni/internal/kube"
	"rbati-cni/internal/load"
	"rbati-cni/internal/network"
	"rbati-cni/internal/utils"
	"time"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var configJSON *load.CNIConfig

const (
	vxlanName 	= "rbati.1"
	bridgeName	= "rbati.0"
)

func init() {

	configJSON = load.LoadFromFile("/etc/cni/net.d/10-rbati.conf")

}

func main() {

	log.Printf("[INFO] Controller starting on node: %s\n", kube.GetNodeName())

	bridge := bridgeName

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
	err = network.SetupBridge(bridge, gatewayIP)
	if err != nil {
		log.Printf("[ERROR] Failed to create Bridge device: %v\n", err)
		os.Exit(1)
	}



	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		log.Println("[INFO] Sync complete.")
		<-ticker.C
	}

}