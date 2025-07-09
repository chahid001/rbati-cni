package controller

import (
	"log"
	// "net"
	// "rbati-cni/internal/kube"

	// "github.com/vishvananda/netlink"
	// v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)



type Controller struct {
	
	clientSet 	*kubernetes.Clientset
	nodeName	string
	Vxlan 		string
}


func NewController(_clientSet *kubernetes.Clientset, _nodeName string, _Vxlan string) *Controller {
	
	return &Controller{
		clientSet: _clientSet,
		nodeName: _nodeName,
		Vxlan: _Vxlan,
	}

}

func (c *Controller) Run(stopCh chan struct{}) {

	factory := informers.NewSharedInformerFactory(c.clientSet, 0)

	nodeInformer := factory.Core().V1().Nodes()

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.onNodeAddfunc,
	})

	factory.Start(stopCh)
}

func (c *Controller) onNodeAddfunc(obj interface{}) {

	log.Println("Im inside OnNodeAdd")
	// node := obj.(*v1.Node)

	// if node.Name == c.nodeName {
	// 	return
	// }

	// internalIP := kube.GetInternalIP(node) // Get internal IP
	// log.Printf("[INFO] Found Internal IP: %v\n", internalIP)

	// _, peerSubnet, err := net.ParseCIDR(node.Spec.PodCIDR) // Get POD CIDR
	// if err != nil {
	// 	log.Printf("[ERROR] Couldn't get POD CIDR: %v", err)
	// }
	// log.Printf("[INFO] Found PeerSubnet: %v\n", peerSubnet.String())

	// vxlanMAC, err := net.ParseMAC(node.Annotations["cni.rbati.io/vxlan-mac"]) // Get VXLAN MAC
	// if err != nil {
	// 	log.Printf("[ERROR] Couldn't parse VXLAN mac address: %v", err)
	// }
	// log.Printf("[INFO] Found VXLAN MAC: %v\n", vxlanMAC.String())


	// vx, err := netlink.LinkByName(c.Vxlan)
	// if err == nil {
	// 	log.Println("[INFO] VXLAN interface already exists")
	// 	return
	// }

	// //---------------- Routing -------------

	// route := netlink.Route{
	// 	LinkIndex: vx.Attrs().Index,
	// 	Dst: peerSubnet,
	// 	Scope: netlink.SCOPE_LINK,
	// }
	
	// if err := netlink.RouteAdd(&route); err != nil {
	// 	log.Printf("[ERROR] Failed to add route: %v", err)
	// 	return
	// }
	// log.Println("Route added successfully.")

	// //---------------- FDB -------------

	// neighFDB := netlink.Neigh{
	// 	LinkIndex: 		vx.Attrs().Index,
	// 	Family:       	netlink.FAMILY_V4,
	// 	State:        	netlink.NUD_PERMANENT,
	// 	Flags:        	netlink.NTF_SELF,
	// 	Type:         	netlink.NDA_DST,
	// 	IP: 			internalIP,
	// 	HardwareAddr: 	vxlanMAC,	
	// }

	// if err := netlink.NeighAdd(&neighFDB); err != nil {
	// 	log.Printf("[ERROR] Failed to add FDB entry: %v", err)
	// 	return
	// }
	// log.Println("FDB entry added successfully.")

}
