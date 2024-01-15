package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	http.HandleFunc("/ping", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("ping---", request.ContentLength)
		writer.WriteHeader(200)
		fmt.Println(writer.Write([]byte("pone")))
	})
	http.HandleFunc("/ping2", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("ping2---", request.ContentLength)
		writer.WriteHeader(200)
		fmt.Println(writer.Write([]byte("pone2")))
	})
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()
	time.Sleep(time.Second * 2)
	var data = `GET /ping HTTP/1.1
Host: localhost:8080
User-Agent: Go-http-client/1.1
Content-Length: 90000

[   data]`
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	go func() {
		var i int
		go func() {
			for {
				if i != 3 {
					continue
				}
				_, err = conn.Write([]byte(strings.Replace(data, "Content-Length: 90000", "Content-Length: 9", 1)))

			}
		}()
		for {
			i++
			fmt.Println("request index", i)
			_, err = conn.Write([]byte(data))
			if err != nil {
				fmt.Println(err)
				return
			}
			time.Sleep(time.Second)
		}
	}()
	var rbuf = make([]byte, 100)
	for {
		n, err := conn.Read(rbuf)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("receive: ", string(rbuf[:n]))
	}

}

func udpClient() {
	conn, err := net.Dial("udp", "39.101.193.248:80")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("connected ------")
	defer conn.Close()
	_, err = conn.Write([]byte("hello"))
	if err != nil {
		panic(err)
	}
	var buf = make([]byte, 2)
	n, err := conn.Read(buf)
	if err != nil {
		panic(err)
	}
	fmt.Println("receive: ", string(buf[:n]))
}

func udpServer() {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   []byte{0, 0, 0, 0},
		Port: 80,
		Zone: "",
	})
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	var buf = make([]byte, 1024)
	log.Println("listen success------")
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println(err)
			return
		}

		_, err = conn.WriteTo(buf[:n], addr)
		if err != nil {
			fmt.Println(err)
		}
	}
}
