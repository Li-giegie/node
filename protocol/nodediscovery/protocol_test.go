package nodediscovery

import (
	"fmt"
	"github.com/Li-giegie/node"
	"sync"
	"testing"
)

func TestNodeDiscovery_BFS(t *testing.T) {
	p := NodeDiscovery{
		node: &node.Server{
			//SrvConf: &node.SrvConf{
			//	Identity: &node.Identity{
			//		Id: 1,
			//	},
			//},
		},
		nodeTab: &NodeTable{
			Cache: make(map[uint32]map[uint32]int64),
			l:     sync.RWMutex{},
		},
		existCache: NewClearMap(),
	}
	p.nodeTab.AddNode(1, 2, 0)
	p.nodeTab.AddNode(1, 3, 0)
	p.nodeTab.AddNode(2, 1, 0)
	p.nodeTab.AddNode(3, 1, 0)
	p.nodeTab.AddNode(3, 4, 0)
	p.nodeTab.AddNode(4, 3, 0)
	p.nodeTab.AddNode(4, 5, 0)
	p.nodeTab.AddNode(5, 4, 0)
	fmt.Println(p.BFS(1))
}
