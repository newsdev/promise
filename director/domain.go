package director

import (
	"errors"
	"strings"
)

type domain struct {
	pathPrefixesList []string
	pathPrefixes     map[string]string
}

func newDomain() *domain {
	return &domain{
		pathPrefixes: make(map[string]string),
	}
}

func (d *domain) setPrefix(prefix, service string) {

	// We only want to add this value to the list if we haven't seen it before.
	if _, ok := d.pathPrefixes[prefix]; !ok {

		// Save a temporary reference to the list and create a new list.
		tmpPathPrefixesList := d.pathPrefixesList
		d.pathPrefixesList = make([]string, len(d.pathPrefixesList)+1)

		// Find the correct index for the prefix, copying all values up to that point.
		i := 0
		for ; i < len(tmpPathPrefixesList) && len(tmpPathPrefixesList[i]) > len(prefix); i++ {
			d.pathPrefixesList[i] = tmpPathPrefixesList[i]
		}

		// Set the prefix.
		d.pathPrefixesList[i] = prefix

		// Copy the remaining values from the old list.
		for ; i < len(tmpPathPrefixesList); i++ {
			d.pathPrefixesList[i+1] = tmpPathPrefixesList[i]
		}
	}

	// Add the prefix/service pair to the map. This needs to be done even if the
	// prefix was previously accounted for.
	d.pathPrefixes[prefix] = service
}

func (d *domain) removePrefix(prefix string) {

	// We only need to remove a prefix if it exists in the map.
	if _, ok := d.pathPrefixes[prefix]; ok {

		// Save a temporary reference to the list and create a new list.
		tmpPathPrefixList := d.pathPrefixesList
		d.pathPrefixesList = make([]string, len(d.pathPrefixesList)-1)

		// Find the index of the prefix we want to remove.
		i := 0
		for ; tmpPathPrefixList[i] != prefix; i++ {
			d.pathPrefixesList[i] = tmpPathPrefixList[i]
		}

		// Copy the remaining prefixes, overwriting the one we want to remove.
		for ; i < len(d.pathPrefixesList); i++ {
			d.pathPrefixesList[i] = tmpPathPrefixList[i+1]
		}

		// Remove the prefix from the map.
		delete(d.pathPrefixes, prefix)
	}
}

func (d *domain) pick(path string) (string, error) {

	// The list of path prefixes is in reverse order by string length. We want
	// to return the first (most specific) match we come accross.
	for _, prefix := range d.pathPrefixesList {
		if strings.HasPrefix(path, prefix) {
			return d.pathPrefixes[prefix], nil
		}
	}

	return "", errors.New("no matching prefix")
}
