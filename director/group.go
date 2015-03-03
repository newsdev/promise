package director

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
)

type group struct {
	ref   map[string]*net.TCPAddr
	addrs []*net.TCPAddr
}

func newGroup() *group {
	return &group{
		ref:   make(map[string]*net.TCPAddr),
		addrs: make([]*net.TCPAddr, 0),
	}
}

func (g *group) refigure() {
	g.addrs = make([]*net.TCPAddr, 0, len(g.ref))
	for _, addr := range g.ref {
		g.addrs = append(g.addrs, addr)
	}
}

func (g *group) set(name, addr string) error {

	// Parse the address as a tcp address.
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	// Check that a valid port was provided.
	if tcpAddr.Port <= 0 {
		return errors.New(fmt.Sprintf("invalid port %d", tcpAddr.Port))
	}

	g.ref[name] = tcpAddr
	g.refigure()
	return nil
}

func (g *group) delete(name string) {
	delete(g.ref, name)
	g.refigure()
}

func (g *group) pick() (*net.TCPAddr, error) {
	n := len(g.addrs)
	switch {
	case n == 0:
		return nil, errors.New("no available backend")
	case n == 1:
		return g.addrs[0], nil
	}

	return g.addrs[rand.Intn(n)], nil
}
