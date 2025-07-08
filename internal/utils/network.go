package utils

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"rbati-cni/internal/logs"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)


func ProducerConsumer(PodSubent string) error {

	path := "/run/rbati-cni"
	filename := "rbati.env"

	fullpath := filepath.Join(path, filename)

	if err := os.MkdirAll(path, 0755); err != nil {
		log.Printf("[ERROR] Failed to create path: %v\n", err)
		return err 
	}

	content := []byte(PodSubent)

	if err := os.WriteFile(fullpath, content, 0644); err != nil {
		log.Printf("[ERROR] Failed to create ENV file: %v\n", err)
		return err 		
	}


	log.Println("[INFO] Created ENV file !")

	return nil

}

func GetMAC(Link string) net.HardwareAddr{

	link, err := netlink.LinkByName(Link)

	if err != nil {
		log.Printf("[ERROR] Failed to create ENV find Link: %v\n", err)
	}

	return link.Attrs().HardwareAddr
}


func GetPodIP(NetnsPath, PodVethEndName string) ([]netlink.Addr, error) {

	podNS, err := netns.GetFromPath(NetnsPath)
	if err != nil {
		logs.Log.Error("Couldn't Open Pod Namespace.", "ERROR", err)
		return nil, err
	}
	defer podNS.Close()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origNS, err := netns.Get()
	if err != nil {
		logs.Log.Error("Couldn't Save the current namespace.", "ERROR", err)
		return nil, err
	}
	defer origNS.Close()

	if err := netns.Set(podNS); err != nil {
		logs.Log.Error("Couldn't switch current context to POD namespace.", "ERROR", err)
		return nil, err
	}

	defer func() {
		netns.Set(origNS)
		logs.Log.Info("Switched back to Host namespace")
	}()

	veth, err := netlink.LinkByName(PodVethEndName)
	if err != nil {
		logs.Log.Error("Couldn't find POD veth end.", "ERROR", err)
		return nil, err

	}

	addrs, err := netlink.AddrList(veth, netlink.FAMILY_V4)

	if err != nil || len(addrs) == 0 {
		logs.Log.Error("Couldn't get IP for POD veth end", "ERROR", err)
		return nil, err
	}

	return addrs, nil
}
