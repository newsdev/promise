package director

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/nytinteractive/srv-proxy/vendor/src/github.com/Sirupsen/logrus"
	"github.com/nytinteractive/srv-proxy/vendor/src/github.com/coreos/go-etcd/etcd"
)

const (
	etcdRoot = "/proxy/"
)

func parseNode(node *etcd.Node) (hostname, backend, value string, fields log.Fields, err error) {

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
	fields = log.Fields{
		"hostname": hostname,
		"backend":  backend,
		"value":    value,
	}
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

func (b *etcdDirector) reset() (uint64, error) {

	// Get the lock for writing.
	b.lock.Lock()
	defer b.lock.Unlock()

	// Get(key string, sort, recursive bool)
	r, err := b.client.Get(etcdRoot, true, true)
	if err != nil {
		return 0, err
	}

	// Clear all of the existing groups.
	log.WithFields(log.Fields{"etcd_index": r.EtcdIndex}).Info("reset")
	b.groups = make(map[string]*group)

	for _, hostNode := range r.Node.Nodes {
		for _, backendNode := range hostNode.Nodes {

			// Break appart the key to get the hostname, backend name, and address
			// value.
			hostname, backend, value, fields, err := parseNode(backendNode)
			log.WithFields(fields).Info("+ backend")
			if err != nil {
				log.WithFields(fields).Warn(err)
				continue
			}

			if err := b.group(hostname).set(backend, value); err != nil {
				log.WithFields(fields).Warn(err)
			}
		}
	}

	return r.EtcdIndex, nil
}

func (b *etcdDirector) watch(index uint64) error {

	waitIndex := index + 1
	for {

		// Issue the watch command. This will block until a new update is
		// available.

		log.WithFields(log.Fields{"wait_index": waitIndex}).Info("watch")
		w, err := b.client.Watch(etcdRoot, waitIndex, true, nil, nil)
		if err != nil {
			return err
		}

		// Update the wait index to insure we don't miss any updates.
		waitIndex = w.EtcdIndex + 1

		// Break appart the key to get the hostname, backend name, and address
		// value.
		hostname, backend, value, fields, err := parseNode(w.Node)
		if err != nil {
			log.WithFields(fields).Warn(err)
			continue
		}

		// Get the lock for writing.
		b.lock.Lock()

		log.WithFields(log.Fields{"etcd_index": w.EtcdIndex}).Info("watch return")
		switch w.Action {
		case "set":
			log.WithFields(fields).Info("+ backend")
			if err := b.group(hostname).set(backend, value); err != nil {
				log.WithFields(fields).Warn(err)
			}
		case "delete", "expire":
			log.WithFields(fields).Info("- backend")
			b.group(hostname).delete(backend)
		}

		// Unlock the lock.
		b.lock.Unlock()
	}
}

func (b *etcdDirector) Watch() error {
	for {

		// Sync the cluster.
		if !b.client.SyncCluster() {
			log.Warn("cluster sync failed")
		}

		// Get the current etcd index value and reset the groups.
		if index, err := b.reset(); err != nil {
			log.Error(err)

			// Sleep in order to not slam etcd with connection attempts.
			time.Sleep(time.Duration(5) * time.Second)
		} else {

			// Wait for updates.
			if err := b.watch(index); err != nil {
				log.Error(err)
			}
		}
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
