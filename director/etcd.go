package director

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/newsdev/promise/vendor/src/github.com/Sirupsen/logrus"
	"github.com/newsdev/promise/vendor/src/github.com/coreos/go-etcd/etcd"
)

const (
	etcdRootKey  = "/proxy/"
	domainsKind  = "domains"
	servicesKind = "services"
)

var (
	undefinedDomainError  = errors.New("domain not defined")
	undefinedServiceError = errors.New("service not defined")
)

type etcdDirector struct {
	client *etcd.Client

	// Access to the routing information is controlled by a read/write mutext.
	lock     sync.RWMutex
	domains  map[string]*domain
	services map[string]*service
}

func NewEtcdDirector(machines []string) *etcdDirector {
	return &etcdDirector{
		client: etcd.NewClient(machines),
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

func (b *etcdDirector) nodeAction(action string, node *etcd.Node) {

	// Check if the node is a directory.
	if node.Dir {

		// Recurse to find leaves.
		for _, next := range node.Nodes {
			b.nodeAction(action, next)
		}
	} else {

		// Parse the components, returning if there are the wrong number.
		key := strings.TrimPrefix(node.Key, etcdRootKey)
		keyComponents := strings.SplitN(key, "/", 3)
		if len(keyComponents) != 3 {
			log.Error(fmt.Sprintf("node key, %s, has the wrong number of components", key))
			return
		}

		// Map the key components and set the field object.
		kind := keyComponents[0]
		name := keyComponents[1]
		detail := keyComponents[2]
		value := node.Value
		fields := log.Fields{
			"kind":   kind,
			"name":   name,
			"detail": detail,
			"value":  value,
		}

		// Switch on the first value, kind.
		switch kind {

		// Process a domain-related action.
		case domainsKind:
			d := b.getDomain(name)
			switch action {
			case "get", "set":
				d.setPrefix(detail, value)
				log.WithFields(fields).Info("+ domain prefix")
			case "delete", "expire":
				d.removePrefix(detail)
				log.WithFields(fields).Info("- domain prefix")
			}

		// Process a service-related action.
		case servicesKind:
			s := b.getService(name)
			switch action {
			case "get", "set":

				// Parse the address.
				addr, err := net.ResolveTCPAddr("tcp", value)
				if err != nil {
					log.WithFields(fields).Error(err)
					return
				}

				if addr.Port <= 0 {
					log.WithFields(fields).Error("invalid port")
					return
				}

				s.setAddr(detail, addr)
				log.WithFields(fields).Info("+ service addr")

			case "delete", "expire":
				s.removeAddr(detail)
				log.WithFields(fields).Info("- service addr")
			}

		default:
			log.WithFields(fields).Error("unknown kind")
		}
	}
}

func (b *etcdDirector) reset() (uint64, error) {

	// Get(key string, sort, recursive bool)
	r, err := b.client.Get(etcdRootKey, true, true)
	if err != nil {
		return 0, err
	}

	// Get the lock for writing.
	b.lock.Lock()

	// Clear current domain and service values.
	b.domains = make(map[string]*domain)
	b.services = make(map[string]*service)

	// Process the node action while holding the lock.
	b.nodeAction(r.Action, r.Node)

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
		r, err := b.client.Watch(etcdRootKey, waitIndex, true, nil, nil)
		if err != nil {
			return err
		}

		// Update the wait index to insure we don't miss any updates.
		waitIndex = r.EtcdIndex + 1

		// Process the node action while holding the lock.
		b.lock.Lock()
		b.nodeAction(r.Action, r.Node)
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
