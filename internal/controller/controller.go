package controller

import (
	"log"
	"net"
	"rbati-cni/internal/kube"

	"github.com/vishvananda/netlink"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)



type Controller struct {
	
	clientSet 		*kubernetes.Clientset
	nodeName		string
	Vxlan 			string
	processedNodes 	map[string]bool
}


func NewController(_clientSet *kubernetes.Clientset, _nodeName string, _Vxlan string) *Controller {
	
	return &Controller{
		clientSet: _clientSet,
		nodeName: _nodeName,
		Vxlan: _Vxlan,
		processedNodes: make(map[string]bool),
	}

}

func (c *Controller) Run(stopCh chan struct{}) {

	factory := informers.NewSharedInformerFactory(c.clientSet, 0)

	nodeInformer := factory.Core().V1().Nodes()

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.onNodeAddfunc,
		UpdateFunc: c.onNodeUpdatefunc,
	})

	factory.Start(stopCh)
    log.Println("[INFO] Waiting for informer caches to sync...")
    if !cache.WaitForCacheSync(stopCh, nodeInformer.Informer().HasSynced) {
        log.Fatalln("failed to sync caches")
    }
    log.Println("[INFO] Informer caches synced. Controller is running.")
	<- stopCh
}

func (c *Controller) proceedNode(node *v1.Node) {

	if node.Name == c.nodeName {
		return
	}

	internalIP := kube.GetInternalIP(node) // Get internal IP
	log.Printf("[INFO] Found Internal IP: %v\n", internalIP)

	_, peerSubnet, err := net.ParseCIDR(node.Spec.PodCIDR) // Get POD CIDR
	if err != nil {
		log.Printf("[ERROR] Couldn't get POD CIDR: %v", err)
	}
	log.Printf("[INFO] Found PeerSubnet: %v\n", peerSubnet.String())

	vxlanMAC, err := net.ParseMAC(node.Annotations["cni.rbati.io/vxlan-mac"]) // Get VXLAN MAC
	if err != nil {
		log.Printf("[ERROR] Couldn't parse VXLAN mac address: %v", err)
	}
	log.Printf("[INFO] Found VXLAN MAC: %v\n", vxlanMAC.String())


	vx, err := netlink.LinkByName(c.Vxlan)
	if err == nil {
		log.Println("[INFO] VXLAN interface already exists")
		return
	}

	//---------------- Routing -------------

	route := netlink.Route{
		LinkIndex: vx.Attrs().Index,
		Dst: peerSubnet,
		Scope: netlink.SCOPE_LINK,
	}
	
	if err := netlink.RouteAdd(&route); err != nil {
		log.Printf("[ERROR] Failed to add route: %v", err)
		return
	}
	log.Println("Route added successfully.")

	//---------------- FDB -------------

	neighFDB := netlink.Neigh{
		LinkIndex: 		vx.Attrs().Index,
		Family:       	netlink.FAMILY_V4,
		State:        	netlink.NUD_PERMANENT,
		Flags:        	netlink.NTF_SELF,
		Type:         	netlink.NDA_DST,
		IP: 			internalIP,
		HardwareAddr: 	vxlanMAC,	
	}

	if err := netlink.NeighAdd(&neighFDB); err != nil {
		log.Printf("[ERROR] Failed to add FDB entry: %v", err)
		return
	}
	c.processedNodes[node.Name] = true
	log.Println("FDB entry added successfully.")

}

func (c *Controller) onNodeAddfunc(obj interface{}) {

	node := obj.(*v1.Node)

	if !c.processedNodes[node.Name]  && isNodeReady(node){
		log.Println("Im inside OnNodeAdd")
		c.proceedNode(node)
	}

}

/*
	When a node is added, it not garantied it will be ready
	so the onNodeAdd will be called, but i can't perform nothing
	because it does not have yet all the info i need.
	With onNodeUpdate i can return untill the node is ready to process the task.
*/

func isNodeReady(node *v1.Node) bool {

	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false

}

func (c *Controller) onNodeUpdatefunc(oldObj, newObj interface{}) { 

	oldNode := oldObj.(*v1.Node)
	newNode := newObj.(*v1.Node)

	oldReady := isNodeReady(oldNode)
	newReady := isNodeReady(newNode)

	if !oldReady && newReady {
		log.Printf("Node %s transitioned to Ready state. Processing.", newNode.Name)
		log.Println("Im inside OnNodeUpdate")
		c.proceedNode(newNode)
	}

}
