package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"strings"

	"github.com/tsenart/patrol"
)

func main() {
	cmd := patrol.Command{
		Log:              log.New(os.Stderr, "", log.LstdFlags),
		APIAddr:          "0.0.0.0:8080",
		ReplicatorAddr:   "0.0.0.0:16000",
		ClusterDiscovery: "static",
	}

	fs := flag.NewFlagSet("patrol", flag.ExitOnError)
	fs.StringVar(&cmd.APIAddr, "api-addr", cmd.APIAddr, "Address to bind HTTP API to")
	fs.StringVar(&cmd.ReplicatorAddr, "replicator-addr", cmd.ReplicatorAddr, "Address to bind replication server to")
	fs.StringVar(&cmd.ClusterDiscovery, "cluster-discovery", cmd.ClusterDiscovery, "Cluster discovery [static]")
	fs.Var(&nodeFlag{nodes: &cmd.ClusterNodes}, "cluster-node", "Cluster node to broadcast to")

	fs.Parse(os.Args[1:])

	if err := cmd.Run(context.Background()); err != nil {
		cmd.Log.Fatal(err)
	}
}

// nodeFlag implements the flag.Value interface for defining node addresses.
type nodeFlag struct{ nodes *[]string }

func (f *nodeFlag) Set(node string) error {
	_, _, err := net.SplitHostPort(node)
	if err != nil {
		return err
	}
	*f.nodes = append(*f.nodes, node)
	return nil
}

func (f *nodeFlag) String() string {
	if f.nodes == nil {
		return ""
	}
	return strings.Join(*f.nodes, ", ")
}
