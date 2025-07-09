package kube

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)


type PeerInfo struct {
	Name   string
	IP     string
	Subnet string
}


func GetNodeName() string {

	return os.Getenv("NODE_NAME") // I will pass it in DeamonSet Manifest

}


func GetInternalIP(node *corev1.Node) net.IP {

	for _, addr := range node.Status.Addresses {

		if addr.Type == corev1.NodeInternalIP {
			return net.ParseIP(addr.Address)
		}

	}

	return nil
}

func GetNodePodCIDR(clientset kubernetes.Interface, nodeName string) (string, error) {
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, v1.GetOptions{})
	if err != nil {
		log.Printf("failed to get node object: %v", err)
		return "", err
	}
	if node.Spec.PodCIDR == "" {
		log.Printf("failed to get node podCIDR: %v", err)
		return "", err
	}
	return node.Spec.PodCIDR, nil
}


func PatchNode(clientset kubernetes.Interface, nodeName string, macVXLAN net.HardwareAddr) error {

	log.Printf("[INFO] Patching node with VXLAN MAC %v", macVXLAN.String())

	patchData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				"cni.rbati.io/vxlan-mac": macVXLAN.String(),
			},
		},
	}
	patchBytes, _ := json.Marshal(patchData)	

	_, err := clientset.CoreV1().Nodes().Patch(
		context.TODO(), 
		nodeName,
		types.MergePatchType,
		patchBytes,
		v1.PatchOptions{})

	if err != nil {
		log.Printf("failed to get patch node annotaiton: %v", err)
		return err
	}

	log.Printf("[INFO] Patched node annotation !")

	return nil
}