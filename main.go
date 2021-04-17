package modtest

import (
    "sync"
    "io"
    "bytes"
    "fmt"
    "context"
    "encoding/binary"
    "encoding/json"
    "io/ioutil"
    "net/http"

    blocks "github.com/ipfs/go-block-format"
    cid "github.com/ipfs/go-cid"
    ipld "github.com/ipfs/go-ipld-format"
)

type moduleTest struct {
    rtclk sync.Mutex
    rootToCidMap map[cid.Cid][]cid.Cid
}

var md *moduleTest
var once sync.Once

func GetInstance() *moduleTest {
    once.Do(func() {
        md = &moduleTest{}
    })
    return md
}

func SealBlockStart(c cid.Cid) error {
    type startParam struct {
        cid string
    }
    var sp = startParam{
        cid: c.KeyString(),
    }
    sj, _ := json.Marshal(sp)
    _, err := http.Post("http://127.0.0.1:12222/api/v0/storage/seal_start", "application/json", bytes.NewReader(sj))
    if err != nil {
        fmt.Printf("Inform sWorker to start seal error:%s\n", err)
        return err
    }
    return nil
}

func SealBlockEnd(c cid.Cid, s bool, ctx context.Context, dserv ipld.DAGService) error {
    type endParam struct {
        cid string
        success bool
    }
    md := GetInstance()
    var ep = endParam{
        cid: c.KeyString(),
        success: s,
    }
    ej, _ := json.Marshal(ep)
    _, err := http.Post("http://127.0.0.1:12222/api/v0/storage/seal_end", "application/json", bytes.NewReader(ej))
    if err != nil {
        fmt.Printf("Inform sWorker to end seal error:%s\n", err)
    }
    if ! s || err != nil {
        dserv.RemoveMany(ctx, md.Cids(c))
    }
    md.Remove(c)
    return err
}

func SealBlock(c cid.Cid, r io.Reader, fromBS bool) ([]blocks.Block, error) {
    resp, err := http.Post("http://127.0.0.1:12222/api/v0/storage/seal", "application/x-www-form-urlencoded", r)
    if err != nil {
		return nil, err
    }
    body, err := ioutil.ReadAll(resp.Body)
    resp.Body.Close()
    if err != nil {
		return nil, err
    }

    md := GetInstance()
    cidLen := len(c.KeyString())

    var wanted []blocks.Block
    snSlic := body[:4]
    srSlic := body[4:8]
    //fnSlic := body[8:12]
    sn := int(binary.BigEndian.Uint32(snSlic))
    sr := int(binary.BigEndian.Uint32(srSlic))
    //fn := int(binary.BigEndian.Uint32(fnSlic))
    ss := body[12:sr]
    //fs := body[sr:]
    ssItemSize := len(ss) / sn
    //fsItemSize := len(fs) / fn

    // Deal with success
    for i := 0; i < sn; i++ {
        rcs := i * ssItemSize
        rce := rcs + cidLen
        blks := rce
        blke := rce + ssItemSize
        rcSlice := ss[rcs:rce]
        blkSlice := ss[blks:blke]
        rc, _ := cid.Cast(rcSlice)
        block, err := blocks.SNewBlockWithCid(blkSlice, c)
        if err == nil {
            wanted = append(wanted, block)
        }
        if ! fromBS {
            md.Add(rc, c)
        }
    }

    return wanted, nil
}

func (m *moduleTest) Add(root cid.Cid, child cid.Cid) {
    m.rtclk.Lock()
    m.rootToCidMap[root] = append(m.rootToCidMap[root], child)
    m.rtclk.Unlock()
}

func (m *moduleTest) Remove(c cid.Cid) {
    m.rtclk.Lock()
    delete(m.rootToCidMap, c)
    m.rtclk.Unlock()
}

func (m *moduleTest) Cids(c cid.Cid) []cid.Cid {
    var ret []cid.Cid
    m.rtclk.Lock()
    copy(ret, m.rootToCidMap[c])
    m.rtclk.Unlock()

    return ret
}
