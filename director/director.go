package director

import (
	"net"
)

type Director interface {
	Pick(hostname string) (*net.TCPAddr, error)
}
