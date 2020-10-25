// Patch Panel / pacthlink
// MIT License Copyright(c) 2020 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
package main

import (
    "fmt"
    "log"
    "net"
    "os"
    "time"

    "github.com/hshimamoto/go-iorelay"
    "github.com/hshimamoto/go-session"
)

func stream(name, remote, local string) {
    lconn, err := session.Dial(local)
    if err != nil {
	log.Println(err)
	return
    }
    defer lconn.Close()
    rconn, err := session.Dial(remote)
    if err != nil {
	log.Println(err)
	return
    }
    defer rconn.Close()
    rconn.Write([]byte(fmt.Sprintf("CONNECTED %s\r\n", name)))
    log.Printf("%s new stream connected", name)
    time.Sleep(100 * time.Millisecond)
    // relay
    iorelay.RelayWithTimeout(lconn, rconn, 24 * time.Hour)
    log.Printf("%s stream closed", name)
}

func link(name, remote, local string) {
    log.Printf("name: %s link: %s-%s", name, remote, local)

    // try to connect forever
    var conn net.Conn
    for {
	var err error
	conn, err = session.Dial(remote)
	if err == nil {
	    break
	}
	time.Sleep(30 * time.Second)
    }
    defer conn.Close()

    conn.Write([]byte(fmt.Sprintf("LINK %s\r\n", name)))

    log.Printf("%s link established", name)

    // enbale keep alive
    if tcp, ok := conn.(*net.TCPConn); ok {
	tcp.SetKeepAlive(true)
	tcp.SetKeepAlivePeriod(time.Minute)
    }

    running := true

    go func() {
	keepalive := time.Now().Add(time.Minute)
	for running {
	    time.Sleep(10 * time.Second)
	    if time.Now().Before(keepalive) {
		continue
	    }
	    // keep alive
	    if _, err := conn.Write([]byte("KeepAlive\r\n")); err != nil {
		log.Printf("%s keepalive: %v\n", name, err)
		break
	    }
	    // next
	    keepalive = time.Now().Add(time.Minute)
	}
	if running {
	    conn.Close()
	}
	log.Printf("%s keepalive goroutine done", name)
    }()
    for {
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
	    log.Printf("%s read error: %v\n", name, err)
	    break
	}
	if n == 0 {
	    log.Printf("%s read close\n", name)
	    break
	}
	//log.Printf("recv: %v", buf[:n])
	go stream(name, remote, local)
    }
    log.Printf("%s close connection", name)
    running = false
}

func main() {
    if len(os.Args) < 4 {
	log.Fatal("patchlink name remote local")
	return
    }
    log.SetFlags(log.Flags() | log.Lmsgprefix)
    log.SetPrefix(fmt.Sprintf("[%d] ", os.Getpid()))
    name := os.Args[1]
    remote := os.Args[2]
    local := os.Args[3]
    for {
	link(name, remote, local)
	// interval
	time.Sleep(10 * time.Second)
    }
}
