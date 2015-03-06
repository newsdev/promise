package director

import (
	"net"
)

type Director interface {
	Pick(hostname, path string) (*net.TCPAddr, error)
}
