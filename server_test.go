package node

import (
	"fmt"
	"testing"
	"time"
)

const (
	sendApi    = 1
	reqApi     = 2
	forwardApi = 3

	permit = "permit"
	deny   = "deny"

	performance_ReqTestApi = 4
)

func TestNodeServer(t *testing.T) {
	srv := NewServer(DEFAULT_ServerAddress,
		WithSrvGoroutine(100, 200),
		WithSrvId(DEFAULT_ServerID),
		WithSrvConnTimeout(time.Second*3),
		WithSrvAuthentication(func(id uint64, data []byte) (ok bool, reply []byte) {
			fmt.Println("auth handle:", id, string(data), len(data))
			switch string(data) {
			case permit:
				return true, []byte(permit)
			case deny:
				return false, []byte(deny)
			default:
				return false, []byte("Invalid Format")
			}
		},
		),
	)

	defer srv.Shutdown()
	if err := srv.ListenAndServer(); err != nil {
		fmt.Println(err)
	}
}
