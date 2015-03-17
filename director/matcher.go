package director

import (
	"errors"
	"strings"
)

var (
	noMatchingPrefixError = errors.New("no matching prefix")
)

type matcher struct {
	prefixesList []string
	prefixes     map[string]string
}

func NewMatcher() *matcher {
	return &matcher{
		prefixes: make(map[string]string),
	}
}

func (m *matcher) setPrefix(prefix, value string) {

	// We only want to add this value to the list if we haven't seen it before.
	if _, ok := m.prefixes[prefix]; !ok {

		// Save a temporary reference to the list and create a new list that has
		// room for another element.
		tmpPrefixesList := m.prefixesList
		m.prefixesList = make([]string, len(m.prefixesList)+1)

		// Find the correct index for the prefix, copying all values up to that point.
		i := 0
		for ; i < len(tmpPrefixesList) && len(tmpPrefixesList[i]) > len(prefix); i++ {
			m.prefixesList[i] = tmpPrefixesList[i]
		}

		// Set the prefix.
		m.prefixesList[i] = prefix

		// Copy the remaining values from the old list.
		for ; i < len(tmpPrefixesList); i++ {
			m.prefixesList[i+1] = tmpPrefixesList[i]
		}
	}

	m.prefixes[prefix] = value
}

// removeServicePrefix removes a prefix from the domain.
func (m *matcher) removePrefix(prefix string) {

	// We only need to remove a prefix if it exists in the map.
	if _, ok := m.prefixes[prefix]; ok {

		// Save a temporary reference to the list and create a new list.
		tmpPrefixesList := m.prefixesList
		m.prefixesList = make([]string, len(m.prefixesList)-1)

		// Find the index of the prefix we want to remove.
		i := 0
		for ; tmpPrefixesList[i] != prefix; i++ {
			m.prefixesList[i] = tmpPrefixesList[i]
		}

		// Copy the remaining prefixes, overwriting the one we want to remove.
		for ; i < len(m.prefixesList); i++ {
			m.prefixesList[i] = tmpPrefixesList[i+1]
		}

		// Remove the prefix from the map.
		delete(m.prefixes, prefix)
	}
}

func (m *matcher) match(path string) (string, error) {

	// The list of path prefixes is in reverse order by string length. We want
	// to return the first (most specific) match we come accross.
	for _, prefix := range m.prefixesList {
		if strings.HasPrefix(path, prefix) {
			return m.prefixes[prefix], nil
		}
	}

	return "", noMatchingPrefixError
}
