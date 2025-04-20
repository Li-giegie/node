package routemanager

import (
	"fmt"
	"github.com/Li-giegie/node/pkg/router"
	"testing"
)

func TestRouter(t *testing.T) {
	var ok bool
	r := NewRouter()
	ok = r.AddRoute(5, 2, 4, 1, []*router.RoutePath{
		{Id: 1, UnixNano: 1},
		{Id: 2, UnixNano: 1},
		{Id: 3, UnixNano: 1},
		{Id: 4, UnixNano: 1},
		{Id: 5, UnixNano: 1},
	})
	fmt.Println(ok)
	ok = r.AddRoute(5, 3, 4, 0, []*router.RoutePath{
		{Id: 1, UnixNano: 1},
		{Id: 2, UnixNano: 1},
		{Id: 3, UnixNano: 1},
		{Id: 4, UnixNano: 1},
		{Id: 5, UnixNano: 1},
	})
	fmt.Println(ok)
	ok = r.AddRoute(4, 1, 3, 1, []*router.RoutePath{
		{Id: 1, UnixNano: 1},
		{Id: 2, UnixNano: 1},
		{Id: 3, UnixNano: 1},
		{Id: 4, UnixNano: 1},
	})
	fmt.Println(ok)
	ok = r.AddRoute(3, 2, 2, 1, []*router.RoutePath{
		{Id: 1, UnixNano: 1},
		{Id: 2, UnixNano: 1},
		{Id: 3, UnixNano: 1},
	})
	fmt.Println(ok)
	ok = r.AddRoute(2, 3, 1, 1, []*router.RoutePath{
		{Id: 1, UnixNano: 1},
		{Id: 2, UnixNano: 1},
	})
	fmt.Println(ok)

	fmt.Println("RemoveRoute", r.RemoveRoute(2, 0))
	fmt.Println("RemoveRoute", r.RemoveRoute(2, 1))

	fmt.Println("RemoveRouteWithPath", r.RemoveRouteWithPath(4, 0))
	fmt.Println("RemoveRouteWithPath", r.RemoveRouteWithPath(4, 1))

	r.RangeRoute(func(empty *router.RouteEmpty) bool {
		fmt.Println("empty", empty.Dst, empty.Via, empty.Hop, empty.UnixNano, empty.Paths)
		return true
	})
	fmt.Println("RemoveRouteWithVia", r.RemoveRouteWithVia(2, 0))
	fmt.Println("RemoveRouteWithVia", r.RemoveRouteWithVia(2, 1))

}
