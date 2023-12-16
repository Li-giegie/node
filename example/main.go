package main

import (
	"time"
)

func main() {
	go Server()
	time.Sleep(time.Second)
	go HandlerClient()
	time.Sleep(time.Second)
	go Client()
	time.Sleep(time.Second)
	select {}
}
