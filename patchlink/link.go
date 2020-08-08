// Patch Panel / pacthlink
// MIT License Copyright(c) 2020 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
package main

import (
    "log"
    "os"
    "time"

    "github.com/hshimamoto/go-iorelay"
    "github.com/hshimamoto/go-session"
)

func stream(remote, local string) {
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
    rconn.Write([]byte("CONNECTED test\r\n"))
    log.Printf("new stream connected")
    time.Sleep(100 * time.Millisecond)
    // relay
    iorelay.RelayWithTimeout(lconn, rconn, 24 * time.Hour)
    log.Printf("stream closed")
}

func main() {
    if len(os.Args) < 3 {
	log.Fatal("patchlink remote local")
	return
    }
    remote := os.Args[1]
    local := os.Args[2]
    log.Println(remote)
    log.Println(local)
    conn, err := session.Dial(remote)
    if err != nil {
	log.Fatal(err)
	return
    }
    defer conn.Close()
    conn.Write([]byte("LINK test\r\n"))
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
	go stream(remote, local)
    }
}
