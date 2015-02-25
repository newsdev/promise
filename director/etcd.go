package director

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/coreos/go-etcd/etcd"
)

const (
	etcdRoot = "/proxy/"
)

func parseNode(node *etcd.Node) (hostname, backend, value string, err error) {

	// Check if the node is a directory.
	if node.Dir {
		err = errors.New("node is a directory")
		return
	}

	key := strings.TrimPrefix(node.Key, etcdRoot)
	keyComponents := strings.Split(key, "/")
	if len(keyComponents) != 2 {
		err = errors.New(fmt.Sprintf("node key, %s, has the wrong number of components", key))
		return
	}

	hostname = keyComponents[0]
	backend = keyComponents[1]
	value = node.Value
	return
}

type etcdDirector struct {
	client *etcd.Client
	groups map[string]*group
	lock   sync.RWMutex
}

func NewEtcdDirector(machines []string) *etcdDirector {
	return &etcdDirector{
		client: etcd.NewClient(machines),
		groups: make(map[string]*group),
	}
}

func (b *etcdDirector) groupNoCreate(name string) *group {

	// Check if we've seen this hostname.
	if group, ok := b.groups[name]; ok {
		return group
	}

	return nil
}

func (b *etcdDirector) group(name string) *group {

	// Check if we've seen this hostname.
	if group := b.groupNoCreate(name); group != nil {
		return group
	}

	// Create a new group and return it.
	b.groups[name] = newGroup()
	return b.groups[name]
}

func (b *etcdDirector) Watch() error {

	// Get(key string, sort, recursive bool)
	r, err := b.client.Get(etcdRoot, true, true)
	if err != nil {
		return err
	}

	// Get the lock for writing.
	b.lock.Lock()

	for _, hostNode := range r.Node.Nodes {
		for _, backendNode := range hostNode.Nodes {

			// Break appart the key to get the hostname, backend name, and address
			// value.
			hostname, backend, value, err := parseNode(backendNode)
			if err != nil {
				log.Println(err)
				continue
			}

			if err := b.group(hostname).set(backend, value); err != nil {
				log.Println(err)
			}
		}
	}

	// Unlock the lock.
	b.lock.Unlock()

	// Set the initial index.
	waitIndex := r.EtcdIndex + 1
	for {

		// Issue the watch command. This will block until a new update is
		// available.
		w, err := b.client.Watch(etcdRoot, waitIndex, true, nil, nil)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// Update the wait index to insure we don't miss any updates.
		waitIndex = w.EtcdIndex + 1

		// Break appart the key to get the hostname, backend name, and address
		// value.
		hostname, backend, value, err := parseNode(w.Node)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// Get the lock for writing.
		b.lock.Lock()

		switch w.Action {
		case "set":
			if err := b.group(hostname).set(backend, value); err != nil {
				log.Println(err)
			}
		case "delete", "expire":
			b.group(hostname).delete(backend)
		}

		// Unlock the lock.
		b.lock.Unlock()
	}

	return nil
}

func (b *etcdDirector) Pick(hostname string) (*net.TCPAddr, error) {

	// Get the lock for reading and defer it's release.
	b.lock.RLock()
	defer b.lock.RUnlock()

	group := b.groupNoCreate(hostname)
	if group == nil {
		return nil, errors.New(fmt.Sprintf("%s: no backends available", hostname))
	}

	pick, err := group.pick()
	if err != nil {
		return nil, err
	}

	return pick, nil
}
