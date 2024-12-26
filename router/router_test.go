package router

import (
	"fmt"
	"testing"
)

func TestRouter(t *testing.T) {
	var ok bool
	router := NewRouter()
	ok = router.AddRoute(5, 2, 4, 1, []*RoutePath{
		{Id: 1, UnixNano: 1},
		{Id: 2, UnixNano: 1},
		{Id: 3, UnixNano: 1},
		{Id: 4, UnixNano: 1},
		{Id: 5, UnixNano: 1},
	})
	fmt.Println(ok)
	ok = router.AddRoute(5, 3, 4, 0, []*RoutePath{
		{Id: 1, UnixNano: 1},
		{Id: 2, UnixNano: 1},
		{Id: 3, UnixNano: 1},
		{Id: 4, UnixNano: 1},
		{Id: 5, UnixNano: 1},
	})
	fmt.Println(ok)
	ok = router.AddRoute(4, 1, 3, 1, []*RoutePath{
		{Id: 1, UnixNano: 1},
		{Id: 2, UnixNano: 1},
		{Id: 3, UnixNano: 1},
		{Id: 4, UnixNano: 1},
	})
	fmt.Println(ok)
	ok = router.AddRoute(3, 2, 2, 1, []*RoutePath{
		{Id: 1, UnixNano: 1},
		{Id: 2, UnixNano: 1},
		{Id: 3, UnixNano: 1},
	})
	fmt.Println(ok)
	ok = router.AddRoute(2, 3, 1, 1, []*RoutePath{
		{Id: 1, UnixNano: 1},
		{Id: 2, UnixNano: 1},
	})
	fmt.Println(ok)

	fmt.Println("RemoveRoute", router.RemoveRoute(2, 0))
	fmt.Println("RemoveRoute", router.RemoveRoute(2, 1))

	fmt.Println("RemoveRouteWithPath", router.RemoveRouteWithPath(4, 0))
	fmt.Println("RemoveRouteWithPath", router.RemoveRouteWithPath(4, 1))

	router.RangeRoute(func(empty *RouteEmpty) bool {
		fmt.Println("empty", empty.Dst, empty.Via, empty.Hop, empty.UnixNano, empty.Paths)
		return true
	})
	fmt.Println("RemoveRouteWithVia", router.RemoveRouteWithVia(2, 0))
	fmt.Println("RemoveRouteWithVia", router.RemoveRouteWithVia(2, 1))

}
