// Patch Panel / pacthpanel
// MIT License Copyright(c) 2020 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
package main

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "os"
    "runtime"
    "strings"
    "time"

    "github.com/hshimamoto/go-iorelay"
    "github.com/hshimamoto/go-session"
)

var g_links int = 0
var g_streams int = 0

func readline(conn net.Conn) (string, []byte, error) {
    buf := make([]byte, 256)
    n := 0
    for {
	r, err := conn.Read(buf[n:])
	if err != nil {
	    return "", nil, err
	}
	if r == 0 {
	    return "", nil, fmt.Errorf("%v: no CRLF", conn)
	}
	//log.Printf("read: %v", buf[n:n+r])
	n += r
	if idx := bytes.Index(buf[0:n], []byte{13, 10}); idx >= 0 {
	    line := string(buf[0:idx])
	    rest := []byte(nil)
	    if n > idx+2 {
		rest = buf[idx+2:n]
	    }
	    return line, rest, nil
	}
	if n >= 256 {
	    return "", nil, fmt.Errorf("%v: line too large", conn)
	}
    }
}

func readrestheader(conn net.Conn, rest []byte) ([]byte, error) {
    if len(rest) == 2 && rest[0] == 13 && rest[1] == 10 {
	return []byte{}, nil
    }
    buf := make([]byte, 256)
    n := copy(buf, rest)
    for {
	r, err := conn.Read(buf[n:])
	if err != nil {
	    return nil, err
	}
	if r == 0 {
	    return nil, fmt.Errorf("%v: EOF", conn)
	}
	n += r
	if idx := bytes.Index(buf[0:n], []byte{13, 10, 13, 10}); idx >= 0 {
	    rest := []byte{}
	    if n > idx+4 {
		rest = buf[idx+4:n]
	    }
	    return rest, nil
	}
	if n >= 256 {
	    return nil, fmt.Errorf("%v: header too large", conn)
	}
    }
}

type Link struct {
    Name string
    Conn net.Conn
    Alive bool
    Queue chan chan net.Conn
    NewConn chan net.Conn
}

type PatchPanel struct {
    //Links []*Link
    Links map[string]*Link
}

func stream(front net.Conn, link *Link) {
    log.Printf("new stream %s", link.Name)
    g_streams++
    defer func() { g_streams-- }()
    q := make(chan net.Conn)
    // request new stream
    link.Queue <- q
    back := <-q
    if back == nil {
	return
    }
    defer back.Close()
    log.Printf("stream %s connected", link.Name)
    // relay
    iorelay.RelayWithTimeout(front, back, 24 * time.Hour)
    log.Printf("stream %s closed", link.Name)
}

func (p *PatchPanel)link(conn net.Conn, line string) {
    defer conn.Close()
    g_links++
    defer func() { g_links-- }()
    linex := strings.Split(line, " ")
    linkname := linex[1]
    log.Printf("link %s", linkname)
    link := &Link{ Name: linkname }
    link.Conn = conn
    link.Queue = make(chan chan net.Conn)
    link.NewConn = make(chan net.Conn)
    link.Alive = true
    p.Links[linkname] = link
    // initialized done
    // keep alive
    finish := make(chan bool)
    go func() {
	buf := make([]byte, 256)
	for {
	    r, err := conn.Read(buf)
	    if err != nil {
		break
	    }
	    if r == 0 {
		break
	    }
	    // discard keep alive
	}
	log.Printf("link %s closed", linkname)
	link.Alive = false
	finish <- true
    }()
    // wait command q
    for {
	select {
	case q := <-link.Queue:
	    // request new connection
	    log.Printf("request new stream: %s", linkname)
	    conn.Write([]byte("NEW\r\n"))
	    var backconn net.Conn = nil
	    select {
	    case backconn = <-link.NewConn:
	    case <-time.After(10 * time.Second):
		log.Printf("waiting new connection: timeout")
	    }
	    q <- backconn
	case <-finish:
	}
	if !link.Alive {
	    break
	}
    }
    log.Printf("close link %s", linkname)
}

func (p *PatchPanel)connected(conn net.Conn, line string) {
    linex := strings.Split(line, " ")
    linkname := linex[1]
    log.Printf("connected %s", linkname)
    link, ok := p.Links[linkname]
    if !ok {
	// close here
	conn.Close()
	return
    }
    // make sure alive
    if !link.Alive {
	conn.Close()
	return
    }
    // conn will be closed in stream
    link.NewConn <- conn
}

func (p *PatchPanel)connect(conn net.Conn, line string, rest []byte) {
    defer conn.Close()
    linex := strings.Split(line, " ")
    linkname := linex[1]
    log.Printf("connect to %s and rest %v\n", linex[1], rest)
    hostport := strings.Split(linkname, ":")
    link, ok := p.Links[hostport[0]]
    if !ok {
	log.Printf("unknown link %s\n", linkname)
	conn.Write([]byte("HTTP/1.0 400 Bad Request\r\n\r\n"))
	return
    }
    if !link.Alive {
	log.Printf("link %s is dead\n", linkname)
	conn.Write([]byte("HTTP/1.0 400 Bad Request\r\n\r\n"))
	return
    }
    rest, err := readrestheader(conn, rest)
    if err != nil {
	log.Printf("header error %v\n", err)
	conn.Write([]byte("HTTP/1.0 400 Bad Request\r\n\r\n"))
	return
    }
    // send back Established
    conn.Write([]byte("HTTP/1.0 200 Established\r\n\r\n"))
    // create new stream
    stream(conn, link)
}

func (p *PatchPanel)Handler(conn net.Conn) {
    line, rest, err := readline(conn)
    if err != nil {
	conn.Close()
	return
    }
    // from back
    if strings.Index(line, "LINK ") == 0 {
	p.link(conn, line)
	return
    }
    if strings.Index(line, "CONNECTED ") == 0 {
	p.connected(conn, line)
	return
    }
    // from front
    if strings.Index(line, "CONNECT ") == 0 {
	p.connect(conn, line, rest)
	return
    }
    if tcp, ok := conn.(*net.TCPConn); ok {
	log.Printf("from %s: unknown: %s", tcp.RemoteAddr().String(), line)
    } else {
	log.Printf("unknown: %s", line)
    }
    conn.Close()
}

func memprofiler() {
    prev := ""
    for {
	runtime.GC()
	// get RSS
	rss := "-"
	if out, err := ioutil.ReadFile("/proc/self/status"); err == nil {
	    lines := strings.Split(string(out), "\n")
	    for _, line := range lines {
		w := strings.Fields(line)
		if w[0] == "VmRSS:" {
		    rss = w[1]
		    break
		}
	    }
	}
	stats := fmt.Sprintf(
	    "==== STATS ===> RSS: %s kB | %d links, %d streams, %d goroutines",
	    rss, g_links, g_streams, runtime.NumGoroutine())
	if stats != prev {
	    log.Println(stats)
	    prev = stats
	}
	time.Sleep(time.Minute)
    }
}

func main() {
    addr := ":8800"
    if len(os.Args) > 1 {
	addr = os.Args[1]
    }
    log.Println("start patchpanel")
    go memprofiler()
    p := &PatchPanel{}
    p.Links = make(map[string]*Link)
    serv, err := session.NewServer(addr, p.Handler)
    if err != nil {
	log.Fatal(err)
	return
    }
    serv.Run()
}
