package nodediscovery

import (
	"encoding/json"
	"fmt"
)

const (
	ACTION_GET uint8 = iota
	ACTION_PUSH
	ACTION_DELETE
)

type ProtoMsg struct {
	PId      string
	Action   uint8
	NodeList []uint32
	Routes   []*Route
	Counter  uint8
}

type Route struct {
	Dst uint32
	Hop uint8
}

func (p *ProtoMsg) String() string {
	action := "invalid"
	switch p.Action {
	case ACTION_GET:
		action = "action_GET"
	case ACTION_PUSH:
		action = "action_PUSH"
	case ACTION_DELETE:
		action = "action_DELETE"
	}
	return fmt.Sprintf("pId %s action %s nodeList %#v counter %d", p.PId, action, p.NodeList, p.Counter)
}

func (p *ProtoMsg) Encode() []byte {
	data, _ := json.Marshal(p)
	//data, _ := jeans.Encode(p.PId, p.Action, p.NodeList, p.Counter)
	return data
}

func (p *ProtoMsg) Decode(b []byte) error {
	return json.Unmarshal(b, p)
	//return jeans.Decode(b, &p.PId, &p.Action, &p.NodeList, &p.Counter)
}
