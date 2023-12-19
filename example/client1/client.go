package main

import (
	"flag"
	"github.com/Li-giegie/node"
	"log"
	"time"
)

const (
	clientReqApi  = 100
	clientSendApi = 101
)
const serverAddr = "39.101.193.248:8088"

var srvaddr = flag.String("rip", "127.0.0.1:8088", "remote ip")

func main() {
	flag.Parse()
	Client("0.0.0.0:8989", *srvaddr, 100)
}

func Client(lAddr, rAddr string, id uint64) {
	client := node.NewClient(
		rAddr,
		node.WithClientId(id),
		node.WithClientLocalIpAddr(lAddr),
		node.WithClientKeepAlive(time.Second*5),
	)
	_, err := client.Connect()
	if err != nil {
		log.Println(err)
		return
	}
	defer client.Close(true)
	client.HandleFunc(clientSendApi, func(id uint64, data []byte) (out []byte, err error) {
		log.Println("client send: ", id, string(data))
		return nil, nil
	})
	client.HandleFunc(clientReqApi, func(id uint64, data []byte) (out []byte, err error) {
		log.Println("client req: ", id, string(data))
		return []byte("client req api handle success"), nil
	})
	badApi, err := client.Registration()
	if err != nil {
		log.Println(err, badApi)
		return
	}

	client.Run()
}
