package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "0.0.0.0:8080")
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	for {
		fmt.Println(conn.Write([]byte("hello")))
		time.Sleep(time.Second * 3)
	}
	for {
		_, err = conn.Read(make([]byte, 10))
		if err != nil {
			log.Println(err)
			break
		}
	}

}
