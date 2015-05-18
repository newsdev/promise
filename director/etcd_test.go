package director

import (
	"net"
	"testing"
)

func BenchmarkEtcdDirectorPickSingleMatch(b *testing.B) {

	e := NewEtcdDirector("/promise", []string{})

	e.domains["localhost"] = newDomain()
	e.domains["localhost"].setServicePrefix("/", "service")
	e.services["service"] = newService()

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:4001")
	if err != nil {
		b.Fatal(err)
	}

	e.services["service"].setAddr("1", addr)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		e.Pick("localhost", "/")
	}

}
