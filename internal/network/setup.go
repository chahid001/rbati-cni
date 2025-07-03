package network

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "rbati-cni/api"

	"github.com/vishvananda/netlink"
)

func bridgeCleanUP(bridge netlink.Link, ip string, client pb.IPAMServiceClient, err error) {

	log.Printf("Couldn't assign IP to Bridge interface; %v\n", err)
	log.Printf("Cleaning.......Deleting Bridge Interafce and releasing IP address")
	if err := netlink.LinkDel(bridge); err != nil {
		log.Printf("Couldn't delete Bridge; %v\n", err)
	}	
	_, err = client.ReleaseIP(context.Background(), &pb.ReleaseRequest{
		Ip: ip,
	})
	if err != nil {
		log.Printf("Couldn't delete IP; %v\n", err)
	}

}

func SetupBridge(name, ip , subnet string, client pb.IPAMServiceClient) error {

	log.Printf("[INFO] Creating Bridge %v\n", name)

	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
	}

	if err := netlink.LinkAdd(bridge); err != nil {
        log.Printf("[ERROR] Could not create bridge %s: %v\n", name, err)
        return err
    }

	resp, err := client.AllocateIP(context.Background(), &pb.AllocateRequest{
		Pod: name,
		CandidateIP: ip,
	})

	if err != nil {
		log.Printf("Couldn't Allocate IP for POD; %v\n", err)
		return err
	}

	if err != nil {
		log.Printf("[ERROR] Couldn't Allocate specified IP for Bridge Interface; %v\n", err)
		log.Printf("[ERROR] Cleaning... Deleting Bridge %v\n", name)
		if err := netlink.LinkDel(bridge); err != nil {
			log.Printf("[ERROR] Couldn't delete Bridge; %v\n", err)
		}
		return err
	}

	log.Printf("[INFO] Attached IP Address '%v' to Bridge %v\n", resp.Ip ,name)

	_, cidr, err := net.ParseCIDR(subnet)


	ones, _ := cidr.Mask.Size()
	bridgeCIDR := fmt.Sprintf("%s/%d", resp.Ip, ones)

	addr, err := netlink.ParseAddr(bridgeCIDR)
	if err != nil {
		bridgeCleanUP(bridge, resp.Ip, client, err)
		return err
	}
	log.Println("[INFO] Parsed Address")

	if err := netlink.AddrAdd(bridge, addr); err != nil {
		bridgeCleanUP(bridge, resp.Ip, client, err)
		return err
	}	
	log.Printf("[INFO] Added IP Address '%v' to Bridge %v\n", resp.Ip ,name)
	
	if err := netlink.LinkSetUp(bridge); err != nil {
		bridgeCleanUP(bridge, resp.Ip, client, err)
		return err
	}
	log.Printf("[INFO] Set Bridge %v UP\n", name)
	
	return nil  
}

func SetupVXLAN(name, iface string, vni int) error {

	_, err := netlink.LinkByName(name)
	if err == nil {
		log.Println("[INFO] VXLAN interface already exists")
		return nil
	}
	
	log.Printf("[INFO] Setup VXLAN Based on Interface %v\n", iface)

	ifVeth, err := netlink.LinkByName(iface)
	if err != nil {
		log.Printf("Couldn't Get Iface %v\n", err)
		return err
	}

	log.Printf("[INFO] Looking for default IP on Interface %v\n", iface)
	addr, err := netlink.AddrList(ifVeth, netlink.FAMILY_V4)
	if err != nil {
		log.Printf("Couldn't Get IPs from Iface %v\n", err)
		return err
	}


	vxlan := &netlink.Vxlan{

		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
		VxlanId:      vni,
		VtepDevIndex: ifVeth.Attrs().Index,
		SrcAddr:      addr[0].IP, // I feel betrayed, No the vxlan wont work with NIL
		Port:         4789,
		Learning:     false, // To always sent ARP requests to be up to date
		Proxy:        false,
		L2miss:       false,
		L3miss:       false,
	}

	if err := netlink.LinkAdd(vxlan); err != nil {
		log.Printf("Couldn't add VXLAN interface %v\n", err)
		return err
	}

	if err := netlink.LinkSetUp(vxlan); err != nil {
		log.Printf("Couldn't set VXLAN interface UP %v\n", err)
		if err := netlink.LinkDel(vxlan); err != nil {
			log.Printf("Couldn't delete Bridge %v\n", err)
		}
		return err
	}

	return nil
}
