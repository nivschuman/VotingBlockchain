package networking_connector

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

type ConnectionHandler func(conn net.Conn, initalizer bool)

type Listener struct {
	Ip                net.IP            //ip to listen on
	Port              uint16            //port to listen on
	ConnectionHandler ConnectionHandler //function to handle received connections
	ln                net.Listener
}

func NewListener(ip net.IP, port uint16, connectionHandler ConnectionHandler) *Listener {
	return &Listener{
		Ip:                ip,
		Port:              port,
		ConnectionHandler: connectionHandler,
	}
}

func (listener *Listener) Listen(wg *sync.WaitGroup) {
	address := net.JoinHostPort(listener.Ip.String(), fmt.Sprint(listener.Port))

	var err error
	listener.ln, err = net.Listen("tcp", address)
	if err != nil {
		log.Panicf("|Listener| Failed to start listener on address %s: %v", address, err)
	}

	wg.Add(1)
	go func() {
		for {
			conn, err := listener.ln.Accept()

			if errors.Is(err, net.ErrClosed) {
				log.Printf("|Listener| Stopping listener on address %s", address)
				wg.Done()
				return
			}

			if err != nil {
				continue
			}

			go listener.ConnectionHandler(conn, false)
		}
	}()
}

func (listener *Listener) StopListening() {
	err := listener.ln.Close()
	if err != nil {
		log.Panicf("|Listener| Failed to close listener on address %s: %v", listener.ln.Addr().String(), err)
	}
}
