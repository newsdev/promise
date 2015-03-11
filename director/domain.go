package director

import (
	"errors"
	"strings"
)

var (
	noMathingPrefixError = errors.New("no matching prefix")
)

type pair struct {
	prefix, service string
}

type domain struct {
	prefixesList []*pair
	prefixes     map[string]string
}

func newDomain() *domain {
	return &domain{
		prefixes: make(map[string]string),
	}
}

// setPrefix adds a prefix/service pair to the domain.
func (d *domain) setPrefix(prefix, service string) {

	// We only want to add this value to the list if we haven't seen it before.
	if _, ok := d.prefixes[prefix]; !ok {

		// Save a temporary reference to the list and create a new list that has
		// room for another element.
		tmpPathPrefixesList := d.prefixesList
		d.prefixesList = make([]*pair, len(d.prefixesList)+1)

		// Find the correct index for the prefix, copying all values up to that point.
		i := 0
		for ; i < len(tmpPathPrefixesList) && len(tmpPathPrefixesList[i].prefix) > len(prefix); i++ {
			d.prefixesList[i] = tmpPathPrefixesList[i]
		}

		// Set the prefix.
		d.prefixesList[i] = &pair{prefix, service}

		// Copy the remaining values from the old list.
		for ; i < len(tmpPathPrefixesList); i++ {
			d.prefixesList[i+1] = tmpPathPrefixesList[i]
		}
	}

	// Add the prefix/service pair to the map. This needs to be done even if the
	// prefix was previously accounted for.
	d.prefixes[prefix] = service
}

// removePrefix removes a prefix from the domain.
func (d *domain) removePrefix(prefix string) {

	// We only need to remove a prefix if it exists in the map.
	if _, ok := d.prefixes[prefix]; ok {

		// Save a temporary reference to the list and create a new list.
		tmpPathPrefixList := d.prefixesList
		d.prefixesList = make([]*pair, len(d.prefixesList)-1)

		// Find the index of the prefix we want to remove.
		i := 0
		for ; tmpPathPrefixList[i].prefix != prefix; i++ {
			d.prefixesList[i] = tmpPathPrefixList[i]
		}

		// Copy the remaining prefixes, overwriting the one we want to remove.
		for ; i < len(d.prefixesList); i++ {
			d.prefixesList[i] = tmpPathPrefixList[i+1]
		}

		// Remove the prefix from the map.
		delete(d.prefixes, prefix)
	}
}

func (d *domain) pick(path string) (string, error) {

	// The list of path prefixes is in reverse order by string length. We want
	// to return the first (most specific) match we come accross.
	for _, prefix := range d.prefixesList {
		if strings.HasPrefix(path, prefix.prefix) {
			return prefix.service, nil
		}
	}

	return "", noMathingPrefixError
}
