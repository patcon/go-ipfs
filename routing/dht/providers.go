package dht

import (
	"time"

	peer "github.com/jbenet/go-ipfs/peer"
	u "github.com/jbenet/go-ipfs/util"
)

type ProviderManager struct {
	providers map[u.Key][]*providerInfo
	local     map[u.Key]struct{}
	lpeer     peer.ID
	getlocal  chan chan []u.Key
	newprovs  chan *addProv
	getprovs  chan *getProv
	halt      chan struct{}
	period    time.Duration
}

type addProv struct {
	k   u.Key
	val *peer.Peer
}

type getProv struct {
	k    u.Key
	resp chan []*peer.Peer
}

func NewProviderManager(local peer.ID) *ProviderManager {
	pm := new(ProviderManager)
	pm.getprovs = make(chan *getProv)
	pm.newprovs = make(chan *addProv)
	pm.providers = make(map[u.Key][]*providerInfo)
	pm.getlocal = make(chan chan []u.Key)
	pm.local = make(map[u.Key]struct{})
	pm.halt = make(chan struct{})
	go pm.run()
	return pm
}

func (pm *ProviderManager) run() {
	tick := time.NewTicker(time.Hour)
	for {
		select {
		case np := <-pm.newprovs:
			if np.val.ID.Equal(pm.lpeer) {
				pm.local[np.k] = struct{}{}
			}
			pi := new(providerInfo)
			pi.Creation = time.Now()
			pi.Value = np.val
			arr := pm.providers[np.k]
			pm.providers[np.k] = append(arr, pi)
		case gp := <-pm.getprovs:
			var parr []*peer.Peer
			provs := pm.providers[gp.k]
			for _, p := range provs {
				parr = append(parr, p.Value)
			}
			gp.resp <- parr
		case lc := <-pm.getlocal:
			var keys []u.Key
			for k, _ := range pm.local {
				keys = append(keys, k)
			}
			lc <- keys
		case <-tick.C:
			for k, provs := range pm.providers {
				var filtered []*providerInfo
				for _, p := range provs {
					if time.Now().Sub(p.Creation) < time.Hour*24 {
						filtered = append(filtered, p)
					}
				}
				pm.providers[k] = filtered
			}
		case <-pm.halt:
			return
		}
	}
}

func (pm *ProviderManager) AddProvider(k u.Key, val *peer.Peer) {
	pm.newprovs <- &addProv{
		k:   k,
		val: val,
	}
}

func (pm *ProviderManager) GetProviders(k u.Key) []*peer.Peer {
	gp := new(getProv)
	gp.k = k
	gp.resp = make(chan []*peer.Peer)
	pm.getprovs <- gp
	return <-gp.resp
}

func (pm *ProviderManager) GetLocal() []u.Key {
	resp := make(chan []u.Key)
	pm.getlocal <- resp
	return <-resp
}

func (pm *ProviderManager) Halt() {
	pm.halt <- struct{}{}
}
