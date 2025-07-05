package kube

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/vishvananda/netlink"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	vxlanMacAnnotation = "rbati-cni.schahid.com/vxlan-mac"
	bridgeName         = "rbati.0"
)

func GetNodeName() string {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatal("FATAL: The NODE_NAME environment variable is not set.")
	}
	return nodeName
}

func UpdateFDBEntry(clientset kubernetes.Interface, vxlanName string) error {
	currentNodeName := GetNodeName()
	vxlanLink, err := netlink.LinkByName(vxlanName)
	if err != nil {
		return fmt.Errorf("could not find vxlan device %s: %w", vxlanName, err)
	}

	bridgeLink, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("could not find bridge device %s: %w", bridgeName, err)
	}

	localNode, err := clientset.CoreV1().Nodes().Get(context.TODO(), currentNodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get current node %s: %w", currentNodeName, err)
	}

	if localNode.Annotations == nil || localNode.Annotations[vxlanMacAnnotation] != vxlanLink.Attrs().HardwareAddr.String() {
		log.Println("[INFO] Annotating node with local VXLAN MAC address...")
		if localNode.Annotations == nil {
			localNode.Annotations = make(map[string]string)
		}
		localNode.Annotations[vxlanMacAnnotation] = vxlanLink.Attrs().HardwareAddr.String()
		_, err = clientset.CoreV1().Nodes().Update(context.TODO(), localNode, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update node annotation: %w", err)
		}
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes.Items {
		if node.Name == currentNodeName {
			continue
		}

		remotePodCIDRStr := node.Spec.PodCIDR
		if remotePodCIDRStr == "" {
			log.Printf("[WARN] Node %s has no PodCIDR, skipping", node.Name)
			continue
		}

		remoteVtepMacStr, ok := node.Annotations[vxlanMacAnnotation]
		if !ok {
			log.Printf("[WARN] Node %s has no VXLAN MAC annotation, skipping", node.Name)
			continue
		}

		var remoteVtepIP net.IP
		for _, addr := range node.Status.Addresses {
			if addr.Type == v1.NodeInternalIP {
				remoteVtepIP = net.ParseIP(addr.Address)
				break
			}
		}
		if remoteVtepIP == nil {
			log.Printf("[WARN] Node %s has no InternalIP, skipping", node.Name)
			continue
		}

		_, remotePodCIDR, err := net.ParseCIDR(remotePodCIDRStr)
		if err != nil {
			log.Printf("[ERROR] Failed to parse PodCIDR %s for node %s: %v", remotePodCIDRStr, node.Name, err)
			continue
		}

		route := netlink.Route{
			LinkIndex: bridgeLink.Attrs().Index,
			Scope:     netlink.SCOPE_LINK,
			Dst:       remotePodCIDR,
		}

		if err := netlink.RouteReplace(&route); err != nil {
			log.Printf("[ERROR] Failed to add route for %s: %v", remotePodCIDRStr, err)
		} else {
			log.Printf("[INFO] Ensured route to %s via %s", remotePodCIDRStr, bridgeName)
		}

		remoteVtepMac, err := net.ParseMAC(remoteVtepMacStr)
		if err != nil {
			log.Printf("[ERROR] Failed to parse MAC %s for node %s: %v", remoteVtepMacStr, node.Name, err)
			continue
		}

		fdbEntry := &netlink.Neigh{
			LinkIndex:    vxlanLink.Attrs().Index,
			Family:       syscall.AF_BRIDGE,
			State:        netlink.NUD_PERMANENT,
			Flags:        netlink.NTF_SELF,
			HardwareAddr: remoteVtepMac,
			IP:           remoteVtepIP,
		}

		_ = netlink.NeighDel(fdbEntry)

		if err := netlink.NeighAppend(fdbEntry); err != nil {
			log.Printf("[ERROR] Failed to add FDB entry for node %s using NeighAppend: %v", node.Name, err)
		} else {
			log.Printf("[INFO] Ensured FDB entry for node %s (%s)", node.Name, remoteVtepIP.String())
		}
	}

	return nil
}