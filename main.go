package modtest

import (
    "sync"

    cid "github.com/ipfs/go-cid"
)

type modtest struct {
    rtclk sync.Mutex
    rootToCidMap map[cid.Cid][]cid.Cid
}

var md *modtest
var once sync.Once

func GetInstance() *modtest {
    once.Do(func() {
        md = &modtest{}
    })
    return md
}

func (m *modtest) Add(root cid.Cid, child cid.Cid) {
    m.rtclk.Lock()
    m.rootToCidMap[root] = append(m.rootToCidMap[root], child)
    m.rtclk.Unlock()
}

func (m *modtest) Remove(c cid.Cid) {
    m.rtclk.Lock()
    delete(m.rootToCidMap, c)
    m.rtclk.Unlock()
}
