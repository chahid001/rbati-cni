package main

import (
	"fmt"
	"log"
	"os"
	"rbati-cni/internal/kube"
	"rbati-cni/internal/load"
	"rbati-cni/internal/network"
	"time"

	pb "rbati-cni/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	
	controllerIP := fmt.Sprintf("%s:50051", configJSON.ControllerIP)
	cnx, err := grpc.NewClient(
		controllerIP, 
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Println("Could not connect with IPAM controller")
	}
	defer cnx.Close()
	client := pb.NewIPAMServiceClient(cnx)	

	err = network.SetupBridge(bridge, configJSON.BridgeIP, configJSON.IPAM.Subnet, client)
	if err != nil {
		log.Printf("[ERROR] Failed to create Bridge device: %v\n", err)
		os.Exit(1)
	}

	network.EnableNat(configJSON.IPAM.Subnet, bridge)


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

	err = network.SetupVXLAN(vxlanName, configJSON.IFace, bridge, 1001)
	if err != nil {
		log.Printf("[ERROR] Failed to create VXLAN device: %v\n", err)
		os.Exit(1)
	}

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {

		log.Println("[INFO] Syncing VXLAN peer FDB entries...")
		err := kube.UpdateFDBEntry(clientset, vxlanName)
		if err != nil {
			log.Printf("[ERROR] Failed to update FDB entries: %v\n", err)
		}
		log.Println("[INFO] Sync complete.")
		<-ticker.C
	}

}
