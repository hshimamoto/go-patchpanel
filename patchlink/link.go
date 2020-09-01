// Patch Panel / pacthlink
// MIT License Copyright(c) 2020 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
package main

import (
    "fmt"
    "log"
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
    log.Printf("new stream connected")
    time.Sleep(100 * time.Millisecond)
    // relay
    iorelay.RelayWithTimeout(lconn, rconn, 24 * time.Hour)
    log.Printf("stream closed")
}

func link(name, remote, local string) {
    log.Printf("name: %s link: %s-%s", name, remote, local)
    conn, err := session.Dial(remote)
    if err != nil {
	log.Fatal(err)
	return
    }
    defer conn.Close()
    conn.Write([]byte(fmt.Sprintf("LINK %s\r\n", name)))
    go func() {
	for {
	    // keep alive
	    conn.Write([]byte("KeepAlive\r\n"))
	    time.Sleep(time.Minute)
	}
    }()
    for {
	buf := make([]byte, 256)
	n, _ := conn.Read(buf)
	if n == 0 {
	    break
	}
	//log.Printf("recv: %v", buf[:n])
	go stream(name, remote, local)
    }
}

func main() {
    if len(os.Args) < 4 {
	log.Fatal("patchlink name remote local")
	return
    }
    name := os.Args[1]
    remote := os.Args[2]
    local := os.Args[3]
    link(name, remote, local)
}
