package nodediscovery

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	ACTION_QUERY uint8 = iota
	ACTION_PULL
	ACTION_PUSH
	ACTION_DELETE
)

type ProtoMsg struct {
	Id       uint32
	SrcId    uint32
	Action   uint8
	NodeList []uint32
	Routes   []*Route
	Counter  uint8
	UnixNano int64
}

type Route struct {
	Dst uint32
	Hop uint8
}

func (p *ProtoMsg) String() string {
	action := "invalid"
	switch p.Action {
	case ACTION_QUERY:
		action = "action_QUERY"
	case ACTION_PULL:
		action = "action_PULL"
	case ACTION_PUSH:
		action = "action_PUSH"
	case ACTION_DELETE:
		action = "action_DELETE"
	}
	w := bytes.NewBuffer(nil)
	w.WriteString("[")
	for _, route := range p.Routes {
		w.WriteString(fmt.Sprintf("dst %d hop %d ,", route.Dst, route.Hop))
	}
	w.WriteString("]")
	return fmt.Sprintf("srcId %d Id %d action %s nodeList %v counter %d routes %s", p.SrcId, p.Id, action, p.NodeList, p.Counter, w.String())
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
