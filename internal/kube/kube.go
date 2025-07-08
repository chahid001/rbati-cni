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


func MacForSubnet(subnet *net.IPNet) net.HardwareAddr {
	mac := []byte{0x0a, 0x58, 0x00, 0x00, 0x00, 0x00} 
	mac[2] = subnet.IP.To4()[0]
	mac[3] = subnet.IP.To4()[1]
	mac[4] = subnet.IP.To4()[2]
	mac[5] = subnet.IP.To4()[3]
	return mac
}


func ReconcilePeers(clientset *kubernetes.Clientset, vxlanName string) error {
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
			log.Printf("[WARN] Node %s has no PodCIDR, skipping.", node.Name)
			continue
		}

		peerIP := GetInternalIP(node)
		if peerIP == nil {
			log.Printf("[WARN] Node %s has no InternalIP, skipping.", node.Name)
			continue
		}

		_, peerSubnet, err := net.ParseCIDR(node.Spec.PodCIDR)
		if err != nil {
			log.Printf("[WARN] Failed to parse PodCIDR for node %s: %v", node.Name, err)
			continue
		}
		
		peerMac := MacForSubnet(peerSubnet)

		log.Printf("[INFO] Adding FDB entry for peer %s: MAC %s -> IP %s", node.Name, peerMac, peerIP)
		fdbEntry := netlink.Neigh{
			LinkIndex:    vxlan.Attrs().Index,
			Family:       syscall.AF_BRIDGE, 
			State:        netlink.NUD_PERMANENT,
			HardwareAddr: peerMac,
			IP:           peerIP,
		}
		if err := netlink.NeighSet(&fdbEntry); err != nil {
			log.Printf("[WARN] Failed to add FDB entry for %s: %v", node.Name, err)
			continue
		}

		remoteGatewayIP := make(net.IP, len(peerSubnet.IP))
    	copy(remoteGatewayIP, peerSubnet.IP)
    	remoteGatewayIP[len(remoteGatewayIP)-1]++

		log.Printf("[INFO] Adding ARP entry for peer %s: IP %s -> MAC %s", node.Name, remoteGatewayIP, peerMac)
		arpEntry := netlink.Neigh{
			LinkIndex:    vxlan.Attrs().Index,
			State:        netlink.NUD_PERMANENT,
			Family:   	  syscall.AF_INET,
			IP:           remoteGatewayIP,
			HardwareAddr: peerMac,
		}
		if err := netlink.NeighSet(&arpEntry); err != nil {
			log.Printf("[WARN] Failed to add ARP entry for %s: %v", node.Name, err)
			continue
		}

		log.Printf("[INFO] Adding route for peer %s: Subnet %s -> Gateway %s", node.Name, peerSubnet.String(), remoteGatewayIP.String())
		route := netlink.Route{
			LinkIndex: vxlan.Attrs().Index,
			Dst:       peerSubnet,
			Gw:        remoteGatewayIP,
		}
		if err := netlink.RouteReplace(&route); err != nil {
			log.Printf("[WARN] Failed to add route for %s: %v", node.Name, err)
			continue
		}
	}

	return nil
}
