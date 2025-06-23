package main

import (
	"ft_cni/internal/kube"
	"ft_cni/internal/load"
	"ft_cni/internal/logs"
	"ft_cni/internal/network"
	"log"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)


var configJSON *load.CNIConfig
const vxlanName = "rbati.1"


func init() {
	logs.Init("/tmp/mycni.log")
	logs.Log.Info("Init Function | Initialized Log")

	logs.Log.Info("env check")
	configJSON = load.LoadFromFile("/etc/cni/net.d/10-ftcni.conflist")
	logs.Log.Info("Logjson check")
	


}

func main() {

	log.Printf("[INFO] Controller starting on node: %s\n", kube.GetNodeName())

	network.EnableNat()

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

	err = network.SetupVXLAN(vxlanName, configJSON.IFace, 1001)
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