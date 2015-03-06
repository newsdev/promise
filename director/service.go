package director

import (
	"errors"
	"math/rand"
	"net"
)

type service struct {
	addrsList []*net.TCPAddr
	addrs     map[string]*net.TCPAddr
}

func newService() *service {
	return &service{
		addrs: make(map[string]*net.TCPAddr),
	}
}

func (g *service) refigure() {
	g.addrsList = make([]*net.TCPAddr, 0, len(g.addrs))
	for _, addr := range g.addrs {
		g.addrsList = append(g.addrsList, addr)
	}
}

func (g *service) setAddr(name string, addr *net.TCPAddr) {
	g.addrs[name] = addr
	g.refigure()
}

func (g *service) removeAddr(name string) {
	delete(g.addrs, name)
	g.refigure()
}

func (g *service) pick() (*net.TCPAddr, error) {
	n := len(g.addrsList)
	switch {
	case n == 0:
		return nil, errors.New("no available name")
	case n == 1:
		return g.addrsList[0], nil
	}

	return g.addrsList[rand.Intn(n)], nil
}
