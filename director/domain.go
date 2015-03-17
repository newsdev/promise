package director

type domain struct {
	services *matcher
}

func newDomain() *domain {
	return &domain{
		services: NewMatcher(),
	}
}

// setServicePrefix adds a prefix/service pair to the domain.
func (d *domain) setServicePrefix(prefix, service string) {
	d.services.setPrefix(prefix, service)
}

// removeServicePrefix removes a prefix from the domain.
func (d *domain) removeServicePrefix(prefix string) {
	d.services.removePrefix(prefix)
}

func (d *domain) pick(path string) (string, error) {
	return d.services.match(path)
}
