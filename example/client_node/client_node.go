package main

import (
	"fmt"
	"github.com/Li-giegie/node"
	"log"
	"time"
)

const serverAddr = "39.101.193.248:8080"

func main() {
	fmt.Println(clientNode(serverAddr))
}

func clientNode(addr string) error {
	cli := node.NewClient(addr)
	reply, err := cli.Connect(node.DEFAULT_ServerID, nil)
	if err != nil {
		return err
	}
	defer cli.Close()
	log.Printf("%s\n", reply)
	err = cli.Send(1000, []byte("head shot ~"))
	if err != nil {
		return err
	}
	reply, err = cli.Request(time.Second*3, 1001, []byte("stick together team ~"))
	if err != nil {
		return err
	}
	log.Printf("%s\n", reply)
	return nil
}
