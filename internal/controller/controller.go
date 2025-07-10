package controller

import (
	"log"
	"net"
	"rbati-cni/internal/kube"
	"rbati-cni/internal/network"
	"strings"
	"sync"
	"time"

	"github.com/vishvananda/netlink"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)



type Controller struct {
	
	clientSet 		*kubernetes.Clientset
	nodeName		string
	Vxlan 			netlink.Link
	processedNodes 	map[string]bool
	processedPods 	map[string]bool
	vxlanMacs      	map[string]net.HardwareAddr // Race condition
	vxlanMacsMutex 	sync.RWMutex
}


func NewController(_clientSet *kubernetes.Clientset, _nodeName string, _Vxlan netlink.Link) *Controller {
	
	return &Controller{
		clientSet: _clientSet,
		nodeName: _nodeName,
		Vxlan: _Vxlan,
		processedNodes: make(map[string]bool),
		processedPods: make(map[string]bool),
		vxlanMacs: make(map[string]net.HardwareAddr),
	}

}

func (c *Controller) Run(stopCh chan struct{}) {

	factory := informers.NewSharedInformerFactory(c.clientSet, 30*time.Second)

	nodeInformer 	:= factory.Core().V1().Nodes()
	podInformer 	:= factory.Core().V1().Pods()

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: 	c.onNodeAddfunc,
		UpdateFunc: c.onNodeUpdatefunc,
		DeleteFunc: c.onNodeDelfunc,
	})
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: 	c.onPodAdd,
		UpdateFunc: c.onPodUpdate,
		DeleteFunc: c.onPodDelete,
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

	macString, ok := node.Annotations["cni.rbati.io/vxlan-mac"]
	if !ok || macString == "" {
		log.Printf("[INFO] VXLAN MAC annotation for node %s not found yet. Will process on update.", node.Name)
		return 
	}
    log.Printf("[DEBUG] Raw annotation string being parsed: %q", macString)
    
    cleanMacString := strings.TrimSpace(macString)

	vxlanMAC, err := net.ParseMAC(cleanMacString) // Get VXLAN MAC

	if err != nil {
		log.Printf("[ERROR] Couldn't parse VXLAN mac address: %v", err)
	}
	log.Printf("[INFO] Found VXLAN MAC: %v\n", vxlanMAC.String())

	c.vxlanMacsMutex.Lock()
	c.vxlanMacs[node.Name] = vxlanMAC
	c.vxlanMacsMutex.Unlock()

	//---------------- Routing -------------

	route := netlink.Route{
		LinkIndex: c.Vxlan.Attrs().Index,
		Dst: peerSubnet,
		Scope: netlink.SCOPE_LINK,
	}
	
	if err := netlink.RouteAdd(&route); err != nil {
		log.Printf("[ERROR] Failed to add route: %v", err)
		return
	}
	log.Println("Route added successfully.")

	//---------------- FDB -------------

	if err := network.AddFDBEntry(c.Vxlan.Attrs().Name, vxlanMAC, internalIP, true) ; err != nil { // Netlink equivalent to bridge fdb
		log.Printf("[ERROR] Failed to add FDB entry: %v", err)
		return
	}
	c.processedNodes[node.Name] = true
	log.Println("FDB entry added successfully.")

}


func (c *Controller) proceedPod(pod *v1.Pod) {

	if pod.Spec.NodeName == c.nodeName {
		return
	}

	if pod.Status.Phase != v1.PodRunning || pod.Status.PodIP == "" {
		return
	}

	podIP := net.ParseIP(pod.Status.PodIP)

	c.vxlanMacsMutex.Lock()
	mac, ok := c.vxlanMacs[pod.Spec.NodeName]
	c.vxlanMacsMutex.Unlock()

	if !ok {
		log.Printf("Could not find VXLAN MAC for node %s yet.", pod.Spec.NodeName)
		return
	}


	log.Printf("[DEBUG] Adding neigh for POD IP %s.", podIP.String())
	neighEntry := netlink.Neigh{
		LinkIndex: c.Vxlan.Attrs().Index,
		IP: podIP,
		HardwareAddr: mac,
		Family: netlink.FAMILY_V4,
		State:        netlink.NUD_PERMANENT,		
	}

	_ = netlink.NeighDel(&neighEntry) // Delete any existing Neigh entry for this ip

	if err := netlink.NeighAdd(&neighEntry); err != nil {
		log.Printf("Failed to add neighbor: %v\n", err)
		return
	}
	log.Printf("[DEBUG] Added neigh for POD IP %s with success.", podIP.String())
	c.processedPods[pod.Name] = true

}



func (c *Controller) onNodeAddfunc(obj interface{}) {

	node := obj.(*v1.Node)

	if !c.processedNodes[node.Name]  && isNodeReady(node){
		log.Println("Im inside OnNodeAdd")
		c.proceedNode(node)
	}

}

func (c *Controller) onNodeDelfunc(obj interface{}) {

	node := obj.(*v1.Node)

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

	macString, ok := node.Annotations["cni.rbati.io/vxlan-mac"]
	if !ok || macString == "" {
		log.Printf("[INFO] VXLAN MAC annotation for node %s not found yet. Will process on update.", node.Name)
		return 
	}    
    cleanMacString := strings.TrimSpace(macString)

	vxlanMAC, err := net.ParseMAC(cleanMacString) // Get VXLAN MAC

	if err != nil {
		log.Printf("[ERROR] Couldn't parse VXLAN mac address: %v", err)
	}
	log.Printf("[INFO] Found VXLAN MAC: %v\n", vxlanMAC.String())

	c.vxlanMacsMutex.Lock()
	c.vxlanMacs[node.Name] = vxlanMAC 
	c.vxlanMacsMutex.Unlock()

	//---------------- Routing -------------

	route := netlink.Route{
		LinkIndex: c.Vxlan.Attrs().Index,
		Dst: peerSubnet,
		Scope: netlink.SCOPE_LINK,
	}
	
	if err := netlink.RouteDel(&route); err != nil {
		log.Printf("[ERROR] Failed to delete route: %v", err)
		return
	}
	log.Println("Route added successfully.")

	//---------------- FDB -------------

	if err := network.AddFDBEntry(c.Vxlan.Attrs().Name, vxlanMAC, internalIP, false) ; err != nil { // Netlink equivalent to bridge fdb
		log.Printf("[ERROR] Failed to add FDB entry: %v", err)
		return
	}
	c.vxlanMacsMutex.Lock()
	delete(c.vxlanMacs, node.Name)
	c.vxlanMacsMutex.Unlock()
	delete(c.processedPods, node.Name)
	log.Println("FDB entry deleted successfully.")

}


func (c *Controller) onPodAdd(obj interface{}) {
	pod := obj.(*v1.Pod)

	if c.processedPods[pod.Name] {
		return
	}

	c.proceedPod(pod)
}

func (c *Controller) onPodDelete(obj interface{}) {
	
	pod := obj.(*v1.Pod)	
	
	if pod.Spec.NodeName == c.nodeName {
		return
	}

	if pod.Status.Phase != v1.PodRunning || pod.Status.PodIP == "" {
		return
	}

	podIP := net.ParseIP(pod.Status.PodIP)

	c.vxlanMacsMutex.Lock()
	mac, ok := c.vxlanMacs[pod.Spec.NodeName]
	c.vxlanMacsMutex.Unlock()

	if !ok {
		log.Printf("Could not find VXLAN MAC for node %s yet.", pod.Spec.NodeName)
		return
	}

	neighEntry := netlink.Neigh{
		LinkIndex: 		c.Vxlan.Attrs().Index,
		IP: 			podIP,
		HardwareAddr: 	mac,
		Family: 		netlink.FAMILY_V4,
		State:        	netlink.NUD_PERMANENT,		
	}

	if err := netlink.NeighDel(&neighEntry); err != nil {
		log.Printf("Failed to add neighbor: %v\n", err)
		return
	}
	c.vxlanMacsMutex.Lock()
	delete(c.vxlanMacs, pod.Spec.NodeName)
	c.vxlanMacsMutex.Unlock()
	delete(c.processedPods, pod.Name)
}

func (c *Controller) onPodUpdate(oldObj, newObj interface{}) {
	pod := newObj.(*v1.Pod)
	if c.processedPods[pod.Name] {
		return
	}
	c.proceedPod(pod)
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
	
	newNode := newObj.(*v1.Node)
	if !isNodeReady(newNode) {
		return
	}

	if c.processedNodes[newNode.Name] {
		return
	}
	log.Printf("Node %s transitioned to Ready state. Processing.", newNode.Name)
	log.Println("Im inside OnNodeUpdate")
	c.proceedNode(newNode)

}
