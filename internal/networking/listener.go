package networking

import (
	"fmt"
	"net"
	"sync"
)

type ConnectionHandler func(conn net.Conn)

type Listener struct {
	IP                string            //ip to listen on
	Port              int               //port to listen on
	ConnectionHandler ConnectionHandler //function to handle received connections
}

func (listener *Listener) Listen(wg *sync.WaitGroup) error {
	defer wg.Done()

	address := fmt.Sprintf("%s:%d", listener.IP, listener.Port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		wg.Add(1)
		go listener.handleConnection(conn, wg)
	}
}

func (listener *Listener) handleConnection(conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done() // Ensure goroutine completion is tracked
	defer conn.Close()

	listener.ConnectionHandler(conn)
}
