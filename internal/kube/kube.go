package kube

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/vishvananda/netlink"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func GetInternalIP(node corev1.Node) net.IP {

	for _, addr := range node.Status.Addresses {

		if addr.Type == corev1.NodeInternalIP {
			return net.ParseIP(addr.Address)
		}

	}

	return nil
}



func AddFDBEntry(vxlan netlink.Link, remoteIP net.IP) error {

	neighbor := netlink.Neigh{
		
		LinkIndex: vxlan.Attrs().Index,
		State: netlink.NUD_PERMANENT,
		IP: remoteIP,
		HardwareAddr: net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	return netlink.NeighAdd(&neighbor)
}

func UpdateFDBEntry(clientSet *kubernetes.Clientset, vxlanName string) error {

	nodes, err := clientSet.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})

	if err != nil {
		log.Printf("failed to List Nodes: %v", err)
		return err
	}

	vxlan, err := netlink.LinkByName(vxlanName)
	if err != nil {
		log.Printf("failed to get VXLAN reference: %v", err)
		return err		
	}


	for _, node := range nodes.Items {

		if node.Name == GetNodeName() {
			continue 
		}

		peerIP := GetInternalIP(node)

		if peerIP == nil {
			continue 
		}

		err := AddFDBEntry(vxlan, peerIP)
		
		if err != nil {
			log.Printf("[WARN] Failed to add FDB entry for peer %s: %v\n", peerIP, err)
			return err
		}
		
		log.Printf("[INFO] FDB entry added for peer %s\n", peerIP)
	}

	return nil

}