package director

import (
	// "fmt"
	"testing"
)

var (
	testGroups = []struct{ service, prefix, value string }{
		{"1", "/", "/_"},
		{"2", "/./", "/./_"},
		{"3", "/././", "/././_"},
		{"4", "/./././", "/./././_"},
		{"5", "/././././", "/././././_"},
		{"6", "/./././././", "/./././././_"},
		{"7", "/././././././", "/././././././_"},
		{"8", "/./././././././", "/./././././././_"},
		{"9", "/././././././././", "/././././././././_"},
	}
)

func TestDomain(t *testing.T) {
	d := newDomain()
	for _, group := range testGroups {
		d.setServicePrefix(group.prefix, group.service)
	}

	for _, group := range testGroups {

		service, err := d.pick(group.value)
		if err != nil {
			t.Fatal(err)
		}

		if service != group.service {
			t.Fatal("incorrect service match")
		}
	}
}

func BenchmarkDomainPickEmpty(b *testing.B) {
	d := newDomain()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		d.pick("/foo")
	}
}

func BenchmarkDomainPickSingleMatch(b *testing.B) {
	d := newDomain()
	d.setServicePrefix("/", "service")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		d.pick("/foo")
	}
}

func BenchmarkDomainPickSingleNoMatch(b *testing.B) {
	d := newDomain()
	d.setServicePrefix("/foo", "service")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		d.pick("/")
	}
}

func BenchmarkDomainMultipleMatch(b *testing.B) {
	d := newDomain()
	for _, group := range testGroups {
		d.setServicePrefix(group.prefix, group.service)
	}

	pick := testGroups[len(testGroups)/2].value

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		d.pick(pick)
	}
}

func BenchmarkDomainMultipleNoMatch(b *testing.B) {
	d := newDomain()
	for _, group := range testGroups {
		d.setServicePrefix(group.prefix, group.service)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		d.pick("a")
	}
}

func BenchmarkDomainMultipleNoMatchLong(b *testing.B) {
	d := newDomain()
	for _, group := range testGroups {
		d.setServicePrefix(group.prefix, group.service)
	}

	pick := "a" + testGroups[len(testGroups)-1].value

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		d.pick(pick)
	}
}
