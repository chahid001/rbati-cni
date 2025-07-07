package kube

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

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

func UpdateRoutes(clientset *kubernetes.Clientset, vxlanName string) error {
    vxlan, err := netlink.LinkByName(vxlanName)
    if err != nil {
        return fmt.Errorf("failed to get VXLAN reference: %w", err)
    }

    nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})
    if err != nil {
        return fmt.Errorf("failed to list nodes: %w", err)
    }

    for _, node := range nodes.Items {
        if node.Name == GetNodeName() {
            continue 
        }

        if node.Spec.PodCIDR == "" {
            continue 
        }

        _, subnet, err := net.ParseCIDR(node.Spec.PodCIDR)
        if err != nil {
            log.Printf("[WARN] Failed to parse PodCIDR for node %s: %v", node.Name, err)
            continue
        }

        route := &netlink.Route{
            LinkIndex: vxlan.Attrs().Index,
            Scope:     netlink.SCOPE_UNIVERSE,
            Dst:       subnet,
        }

        if err := netlink.RouteReplace(route); err != nil {
            log.Printf("[WARN] Failed to add route for peer %s (%s): %v", node.Name, subnet, err)
            continue
        }
        log.Printf("[INFO] Route added for peer %s -> %s\n", node.Name, subnet)
    }
    return nil
}

func AddFDBEntry(vxlan netlink.Link, remoteIP net.IP) error {
    floodMac, _ := net.ParseMAC("00:00:00:00:00:00")

    return netlink.NeighAppend(&netlink.Neigh{
        LinkIndex:    vxlan.Attrs().Index,
        Family:       syscall.AF_BRIDGE,
        State:        netlink.NUD_PERMANENT,
        Flags:        netlink.NTF_SELF,
        HardwareAddr: floodMac,
        IP:           remoteIP,
    })
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

        if err := AddFDBEntry(vxlan, peerIP); err != nil {
            log.Printf("[WARN] Failed to add FDB entry for peer %s: %v\n", peerIP, err)
            continue
        }
		
		log.Printf("[INFO] FDB entry added for peer %s\n", peerIP)
	}

	return nil

}