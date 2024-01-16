package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	listen, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		log.Println(err)
	}
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go func(tConn net.Conn) {
			defer tConn.Close()
			buf := make([]byte, 1024)
			n := 0
			for {
				err = tConn.SetReadDeadline(time.Now().Add(time.Second * 3))
				fmt.Println(buf[:n], err)
				if n, err = conn.Read(buf); err != nil {
					fmt.Println(err)
					continue
				}

			}
		}(conn)
	}
}
