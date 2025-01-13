package routerbfs

import (
	"fmt"
	"log"
	"reflect"
	"testing"
	"time"
)

func TestDecodeEncode(t *testing.T) {
	syncNode := SyncNode{
		UnixNano: time.Now().UnixNano(),
		RootId:   10,
		SubIds:   []uint32{1, 2, 3},
	}
	data := syncNode.Encode()
	//fmt.Println(data)
	dSyncNode := new(SyncNode)
	fmt.Println(dSyncNode.Decode(data))
	fmt.Println(syncNode, dSyncNode)

	subInfo := SubInfo{
		Id:      123,
		UnixNao: time.Now().UnixNano(),
	}
	data = subInfo.Encode()
	//fmt.Println(data)

	nodeInfo := NodeInfo{
		RootId: 1357,
		SubIds: []SubInfo{
			{
				Id:      100,
				UnixNao: 100123,
			},
			{
				Id:      100,
				UnixNao: 100123,
			},
			{
				Id:      2,
				UnixNao: time.Now().UnixNano(),
			},
		},
	}
	data = nodeInfo.Encode()
	//fmt.Println(data)
	rNodeInfo := new(NodeInfo)
	fmt.Println(rNodeInfo.Decode(data))
	fmt.Println(nodeInfo, rNodeInfo)
}

func TestDecodeSyncNode(t *testing.T) {
	m := ProtoMsg{
		Id:     1,
		Action: 10,
		Paths:  []uint32{1},
		Nodes: []NodeInfo{
			{
				RootId: 100,
				SubIds: []SubInfo{
					{
						Id:      100,
						UnixNao: 100123,
					},
				},
			},
		},
		SyncNode: &SyncNode{
			UnixNano: 1000,
			RootId:   123,
			SubIds:   []uint32{1},
		},
	}
	data := m.Encode()
	fmt.Println(data)
	rm := ProtoMsg{}
	err := rm.Decode(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reflect.DeepEqual(m, rm))
}
