package director

import (
	"fmt"
	"testing"
)

func TestDomain(t *testing.T) {
	d := newDomain()

	d.setPrefix("/blah", "b")
	d.setPrefix("/bl", "b")
	d.setPrefix("/blahhhh", "b")
	d.setPrefix("/bla", "b")

	for _, prefix := range d.pathPrefixesList {
		fmt.Println(prefix)
	}

	d.removePrefix("/blah")
	d.removePrefix("doesnotexist")

	fmt.Println("-")

	for _, prefix := range d.pathPrefixesList {
		fmt.Println(prefix)
	}

	fmt.Println(d.pick("/blah/something"))

}
