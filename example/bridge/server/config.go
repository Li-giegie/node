package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Bridge struct {
	Id      uint32
	Address string
}

type Config struct {
	Id     uint32
	Addr   string
	Key    string
	Bridge []*Bridge
}

var conf Config

func genConfTemplate(name string) {
	conf = Config{
		Id:   1,
		Addr: ":8001",
		Key:  "hello",
		Bridge: []*Bridge{
			&Bridge{
				Id:      1,
				Address: ":8001",
			},
			&Bridge{
				Id:      2,
				Address: ":8001",
			},
		},
	}
	data, err := json.MarshalIndent(conf, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile(name, data, 0666); err != nil {
		log.Fatal(err)
	}
	fmt.Println("gen success")
	os.Exit(0)
}

func loadConfTemplate(name string) {
	data, err := os.ReadFile(name)
	if err != nil {
		log.Fatal(err)
	}
	if err = json.Unmarshal(data, &conf); err != nil {
		log.Fatal(err)
	}
}
