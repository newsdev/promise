package director

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
)

const (
	domainsKind  = "domains"
	servicesKind = "services"
)

var (
	undefinedDomainError  = errors.New("domain not defined")
	undefinedServiceError = errors.New("service not defined")
	nodeComponentsError   = errors.New("node has the wrong number of components")
)

type etcdParsedNode struct {
	kind, name, detail, value string
}

func newParsedNode(etcdRootKey string, node *etcd.Node) (*etcdParsedNode, error) {

	// Parse the key.
	key := strings.TrimPrefix(node.Key, fmt.Sprintf("/%s/", etcdRootKey))
	keyComponents := strings.SplitN(key, "/", 3)
	if len(keyComponents) != 3 {
		return nil, nodeComponentsError
	}

	return &etcdParsedNode{
		kind:   keyComponents[0],
		name:   keyComponents[1],
		detail: keyComponents[2],
		value:  node.Value,
	}, nil
}

func (e *etcdParsedNode) fields() log.Fields {
	return log.Fields{
		"kind":   e.kind,
		"name":   e.name,
		"detail": e.detail,
		"value":  e.value,
	}
}

type etcdDirector struct {
	client      *etcd.Client
	etcdRootKey string

	// Access to the routing information is controlled by a read/write mutext.
	lock     sync.RWMutex
	domains  map[string]*domain
	services map[string]*service
}

func NewEtcdDirector(etcdRootKey string, machines []string) *etcdDirector {
	return &etcdDirector{
		etcdRootKey: etcdRootKey,
		client:      etcd.NewClient(machines),
		domains:     make(map[string]*domain),
		services:    make(map[string]*service),
	}
}

// getDomain will return the domain of the given name. If that domain is
// missing, it will create a new domain, store it, and return it.
func (b *etcdDirector) getDomain(name string) *domain {

	// Check if we've seen this hostname.
	if d, ok := b.domains[name]; ok {
		return d
	}

	// Create a new group and return it.
	b.domains[name] = newDomain()
	return b.domains[name]
}

// getService will return the service of the given name. If that service
// is missing, it will create a new service, store it, and return it.
func (b *etcdDirector) getService(name string) *service {

	// Check if we've seen this hostname.
	if s, ok := b.services[name]; ok {
		return s
	}

	// Create a new group and return it.
	b.services[name] = newService()
	return b.services[name]
}

func (b *etcdDirector) processDomainService(dn, prefix, service string, add bool) {

	// Get the domain and set a map of fields for logging.
	d := b.getDomain(dn)
	fields := log.Fields{
		"dn":      dn,
		"prefix":  prefix,
		"service": service,
	}

	// Check whether or not this action is additive.
	if add {
		d.setServicePrefix(prefix, service)
		log.WithFields(fields).Info("+ domain service prefix")
	} else {
		d.removeServicePrefix(prefix)
		log.WithFields(fields).Info("- domain service prefix")
	}
}

func (b *etcdDirector) processDomainNode(e *etcdParsedNode, add bool) {

	// Parse the detail into components and set the command.
	detailComponents := strings.Split(e.detail, "/")
	command := detailComponents[len(detailComponents)-1]

	// Determine the prefix from the remaining components.
	var prefix string
	if len(detailComponents) > 1 {
		prefix = strings.Join(detailComponents[:len(detailComponents)-2], "/")
	}

	// Switch on the command.
	switch command {
	case ".service":
		b.processDomainService(e.name, prefix, e.value, add)
	default:
		log.WithFields(e.fields()).WithField("command", command).Error("unknown command")
	}
}

func (b *etcdDirector) processServiceAddr(sn, name string, addr *net.TCPAddr, add bool) {

	// Get the domain and set a map of fields for logging.
	s := b.getService(sn)
	fields := log.Fields{
		"sn":   sn,
		"name": name,
		"addr": addr.String(),
	}

	// Check whether or not this action is additive.
	if add {
		s.setAddr(name, addr)
		log.WithFields(fields).Info("+ service addr")
	} else {
		s.removeAddr(name)
		log.WithFields(fields).Info("- service addr")
	}
}

func (b *etcdDirector) processServiceNode(e *etcdParsedNode, add bool) {

	// Parse the address.
	addr, err := net.ResolveTCPAddr("tcp", e.value)
	if err != nil {
		log.WithFields(e.fields()).Error(err)
		return
	}

	// Check for a valid port.
	if add && addr.Port <= 0 {
		log.WithFields(e.fields()).WithField("port", addr.Port).Error("invalid port")
		return
	}

	// Process the service addr.
	b.processServiceAddr(e.name, e.detail, addr, add)
}

func (b *etcdDirector) nodeAction(node *etcd.Node, add bool) {

	// Check if the node is a directory.
	if node.Dir {

		// Recurse to find leaves.
		for _, next := range node.Nodes {
			b.nodeAction(next, add)
		}
	} else {

		parsedNode, err := newParsedNode(b.etcdRootKey, node)
		if err != nil {
			log.Error(err)
		} else {

			// Check what kind of node this is.
			switch parsedNode.kind {
			case domainsKind:
				b.processDomainNode(parsedNode, add)
			case servicesKind:
				b.processServiceNode(parsedNode, add)
			default:
				log.WithFields(parsedNode.fields()).Error("unknown kind")
			}
		}
	}
}

func (b *etcdDirector) reset() (uint64, error) {

	// Get(key string, sort, recursive bool)
	r, err := b.client.Get(b.etcdRootKey, true, true)
	if err != nil {
		return 0, err
	}

	// Get the lock for writing.
	b.lock.Lock()

	// Clear current domain and service values.
	b.domains = make(map[string]*domain)
	b.services = make(map[string]*service)

	// Process the node action while holding the lock.
	b.nodeAction(r.Node, true)

	// Release the lock.
	b.lock.Unlock()

	// Return the etcd index.
	return r.EtcdIndex, nil
}

func (b *etcdDirector) watch(index uint64) error {

	// Wait for the update after the current etcd index.
	waitIndex := index + 1
	for {

		// Issue the watch command. This will block until a new update is
		// available or an error occurs.
		log.WithFields(log.Fields{"wait_index": waitIndex}).Info("watch")
		r, err := b.client.Watch(b.etcdRootKey, waitIndex, true, nil, nil)
		if err != nil {
			return err
		}

		// Update the wait index to insure we don't miss any updates.
		waitIndex = r.EtcdIndex + 1

		// Process the node action while holding the lock.
		b.lock.Lock()

		// Determine if this action is additive or not.
		switch r.Action {
		case "get", "set":
			b.nodeAction(r.Node, true)
		case "delete", "expire":
			b.nodeAction(r.Node, false)
		default:
			log.WithField("action", r.Action).Info("unknown action")
		}

		// Release the lock.
		b.lock.Unlock()
	}
}

// Watch monitors etcd for any configuration updates and applies them to the
// director. It's meant to run consistently and only log errors it encounters.
func (b *etcdDirector) Watch() {
	for {

		// Sync the cluster. We're only issuing a warning on failure, because it's
		// possible that we have sufficient information to read from at least one
		// etcd instance.
		if !b.client.SyncCluster() {
			log.Warn("cluster sync failed")
		}

		// Try to get the current etcd index value and reset the groups.
		if index, err := b.reset(); err != nil {
			log.Error(err)

			// Sleep in order to not slam etcd with connection attempts.
			time.Sleep(time.Duration(5) * time.Second)
		} else {

			// Wait for updates. In the event of an error, we go straight to another
			// reset attempt.
			if err := b.watch(index); err != nil {
				log.Error(err)
			}
		}
	}
}

// Pick attempts to match a hostname/path combination with the address of a
// server that can handle the request.
func (b *etcdDirector) Pick(hostname, path string) (*net.TCPAddr, error) {

	// Get the lock for reading and defer it's release.
	b.lock.RLock()
	defer b.lock.RUnlock()

	domain := b.domains[hostname]
	if domain == nil {
		return nil, undefinedDomainError
	}

	serviceName, err := domain.pick(strings.TrimPrefix(path, "/"))
	if err != nil {
		return nil, err
	}

	service := b.services[serviceName]
	if service == nil {
		return nil, undefinedServiceError
	}

	return service.pick()
}
