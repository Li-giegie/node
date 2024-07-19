package basic

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Li-giegie/node/common"
	"log"
	"os"
	"strings"
	"time"
)

type SendRequestForwardFlagSet struct {
	Time  uint
	DstId uint
	Text  string
	Debug bool
	fSet  *flag.FlagSet
}

func NewSendRequestForward() *SendRequestForwardFlagSet {
	sqf := new(SendRequestForwardFlagSet)
	sqf.fSet = flag.NewFlagSet("", flag.ContinueOnError)
	sqf.fSet.UintVar(&sqf.Time, "time", 3000, "timeout/ms")
	sqf.fSet.BoolVar(&sqf.Debug, "debug", false, "debug print")
	sqf.fSet.UintVar(&sqf.DstId, "id", 0, "dest id")
	sqf.fSet.Usage = func() {
		sqf.fSet.PrintDefaults()
	}
	return sqf
}

func (s *SendRequestForwardFlagSet) Parse(args []string) (err error) {
	err = s.fSet.Parse(args)
	if s.Debug {
		fmt.Printf("[DEBUG] destId: %d, timeout: %d, text: %s, debug: %v\n", s.DstId, s.Time, s.Text, s.Debug)
	}
	s.Text = strings.Join(s.fSet.Args(), " ")
	return
}

type StartFlagSet struct {
	LAddr       string
	LId         uint
	RAddr       string
	Rid         uint
	Key         string
	AuthTimeout uint
	Debug       bool
	fSet        *flag.FlagSet
}

func NewStartFlagSet() *StartFlagSet {
	fSet := new(StartFlagSet)
	fSet.fSet = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fSet.fSet.StringVar(&fSet.LAddr, "laddr", "0.0.0.0:8000", "local addr")
	fSet.fSet.UintVar(&fSet.LId, "lid", 0, "local id")
	fSet.fSet.StringVar(&fSet.RAddr, "raddr", "0.0.0.0:8000", "remote addr")
	fSet.fSet.UintVar(&fSet.Rid, "rid", 0, "local id")
	fSet.fSet.UintVar(&fSet.AuthTimeout, "timeout", 3000, "auth timeout /ms")
	fSet.fSet.StringVar(&fSet.Key, "key", "hello", "auth key")
	fSet.fSet.BoolVar(&fSet.Debug, "debug", false, "debug print")
	return fSet
}
func (s *StartFlagSet) Parse() *StartFlagSet {
	err := s.fSet.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if s.Debug {
		fmt.Printf("[DEBUG] laddr: %s, lid: %d, raddr: %s, rid: %d, key: %s, auth-timeout\\ms: %d, debug: %v\n", s.LAddr, s.LId, s.RAddr, s.Rid, s.Key, s.AuthTimeout, s.Debug)
	}
	return s
}

type Conns interface {
	GetConn(id uint16) (common.Conn, bool)
}

func ParseCmd(conn common.Conn, conns Conns) {
	fset := NewSendRequestForward()
	scan := bufio.NewScanner(os.Stdin)
	var ok bool
	var err error
	fmt.Print(">>")
	for scan.Scan() {
		cmds := strings.Split(scan.Text(), " ")
		ok = true
		switch cmds[0] {
		case "send":
			if err = fset.Parse(cmds[1:]); err != nil {
				fmt.Println("send cmd err", err)
			} else {
				if fset.Text != "" {
					if conn == nil {
						conn, ok = conns.GetConn(uint16(fset.DstId))
					}
					if ok {
						if err = conn.Send([]byte(fset.Text)); err != nil {
							fmt.Println("send err", err)
						}
					} else {
						fmt.Println("conn not exist")
					}
				}
			}
		case "forward":
			if err = fset.Parse(cmds[1:]); err != nil {
				fmt.Println("forward cmd err", err)
			} else {
				if fset.Text != "" {
					func() {
						if conn == nil {
							conn, ok = conns.GetConn(uint16(fset.DstId))
						}
						if ok {
							ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(fset.Time))
							defer cancel()
							data, err := conn.Forward(ctx, uint16(fset.DstId), []byte(fset.Text))
							if err != nil {
								fmt.Println("forward err", err)
								return
							}
							log.Println(string(data))
						} else {
							fmt.Println("conn not exist")
						}
					}()
				}
			}
		case "request":
			if err = fset.Parse(cmds[1:]); err != nil {
				fmt.Println("request cmd err", err)
			} else {
				if fset.Text != "" {
					func() {
						if conn == nil {
							conn, ok = conns.GetConn(uint16(fset.DstId))
						}
						if ok {
							ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(fset.Time))
							defer cancel()
							data, err := conn.Request(ctx, []byte(fset.Text))
							if err != nil {
								fmt.Println("Request err", err)
								return
							}
							log.Println(string(data))
						} else {
							fmt.Println("conn not exist")
						}
					}()
				}
			}
		case "help", "h", "-h":
			fmt.Println("cmd [request -h | send -h | forward -h | exit(q、quit、exit)]")
		case "":
		case "exit", "quit", "q":
			return
		default:
			fmt.Println("未知命令")
		}
		fmt.Print(">>")
	}
}
