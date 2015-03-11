package director

import (
	"net"
	"strconv"
	"testing"
)

func TestService(t *testing.T) {
	s := newService()

	if _, err := s.pick(); err == nil {
		t.Error("no error reported on pick for an empty service")
	}

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:4001")
	if err != nil {
		t.Fatal(err)
	}

	s.setAddr("1", addr)

	receivedAddr, err := s.pick()
	if err != nil {
		t.Fatal(err)
	}

	if receivedAddr != addr {
		t.Error("received the wrong addr")
	}
}

func BenchmarkServicePickEmpty(b *testing.B) {
	s := newService()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.pick()
	}
}

func BenchmarkServicePickSingle(b *testing.B) {
	s := newService()

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:4001")
	if err != nil {
		b.Fatal(err)
	}

	s.setAddr("1", addr)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.pick()
	}
}

func BenchmarkServicePickMultple(b *testing.B) {
	s := newService()

	for i := 0; i < 1024; i++ {

		a := strconv.Itoa(i)

		addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:"+a)
		if err != nil {
			b.Fatal(err)
		}

		s.setAddr(a, addr)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.pick()
	}
}
