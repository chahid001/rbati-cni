package main

import (
	"context"
	"log"
	"net"

	pb "rbati-cni/api"
	internalipam "rbati-cni/internal/ipam"
	"rbati-cni/internal/load"

	"google.golang.org/grpc"
)


var configJSON *load.CNIConfig

func init() {

	configJSON = load.LoadFromFile("/etc/cni/net.d/10-rbati.conf")

}

type ipamServer struct {
	pb.UnimplementedIPAMServiceServer
	store *internalipam.IPAM
}

func (s *ipamServer) AllocateIP(ctx context.Context, req *pb.AllocateRequest) (*pb.AllocateResponse, error) {
	ip, mask, err := s.store.Allocate(req.CandidateIP)
	if err != nil {
		return nil, err
	}
	return &pb.AllocateResponse{
		Ip:   ip.String(),
		Mask: net.IP(mask).String(),
	}, nil
}

func (s *ipamServer) ReleaseIP(ctx context.Context, req *pb.ReleaseRequest) (*pb.ReleaseResponse, error) {
	err := s.store.Release(req.Ip)
	if err != nil {
		return nil, err
	}
	return &pb.ReleaseResponse{}, nil
}

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	ipamDB, err := internalipam.NewIPAM(configJSON.IPAM.Subnet, "/var/lib/cni/central_ipam.db")
	if err != nil {
		log.Fatalf("Failed to start IPAM: %v", err)
	}
	defer ipamDB.Close()

	server := grpc.NewServer()
	pb.RegisterIPAMServiceServer(server, &ipamServer{store: ipamDB})

	log.Println("IPAM controller running on port 50051")
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
