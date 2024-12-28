package routerbfs

import (
	"time"
)

type RouteEmpty struct {
	dst      uint32
	via      uint32
	hop      uint8
	unixNano int64
	fullPath []uint32
}

// Dst 路由目的
func (r *RouteEmpty) Dst() uint32 {
	return r.dst
}

// Via 路由下一跳
func (r *RouteEmpty) Via() uint32 {
	return r.via
}

// Hop 路由跳数，静态路由该值无效
func (r *RouteEmpty) Hop() uint8 {
	return r.hop
}

// Duration 路由更新时间 unixNano时间戳
func (r *RouteEmpty) Duration() time.Duration {
	return time.Duration(r.unixNano)
}

// FullPath 从源到目的的路由路径
func (r *RouteEmpty) FullPath() []uint32 {
	return r.fullPath
}
