package modtest

import (
    "sync"
    "io"
    "bytes"
    "fmt"
    "context"
    "errors"
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

type sessionKV struct {
    sk string
    cid string
}

var (
    sessionKey = "key"
    sworkerBaseUrl = "http://127.0.0.1:12222/api/v0"
)

var md *moduleTest
var once sync.Once

func GetInstance() *moduleTest {
    once.Do(func() {
        md = &moduleTest{}
    })
    return md
}

func undefinedError() error {
    if len(sworkerBaseUrl) == 0 {
        return errors.New("sWorker base url not defined!")
    }
    return nil
}

func unexpectedError() error {
    return errors.New("unexpected error happens!")
}

func getKVFromSession(ctx context.Context) sessionKV {
    buf := ctx.Value(sessionKey)
    var kv sessionKV
    err := json.Unmarshal(buf, &kv)
    if err != nil {
        return err
    }
    return kv
}

func SealBlockStart(ctx context.Context, c cid.Cid) error {
    err := undefinedError()
    if err != nil {
        return err
    }

    type startParam struct {
        sk string
        cid string
    }

    kv := getKVFromSession(ctx)
    if kv == nil {
        return unexpectedError()
    }
    var sp = startParam{
        sk: kv.sk,
        cid: c.String(),
    }
    sj, _ := json.Marshal(sp)
    _, err = http.Post(sworkerBaseUrl + "/storage/seal_start", "application/json", bytes.NewReader(sj))
    if err != nil {
        fmt.Printf("Inform sWorker to start seal error:%s\n", err)
        return err
    }
    return nil
}

func SealBlockEnd(ctx context.Context, c cid.Cid, s bool, dserv ipld.DAGService) error {
    err := undefinedError()
    if err != nil {
        return err
    }

    type endParam struct {
        sk string
        cid string
        success bool
    }

    md := GetInstance()
    kv := getKVFromSession(ctx)
    if kv == nil {
        return unexpectedError()
    }
    var ep = endParam{
        sk: kv.sk,
        cid: c.String(),
        success: s,
    }
    ej, _ := json.Marshal(ep)
    _, err = http.Post(sworkerBaseUrl + "/storage/seal_end", "application/json", bytes.NewReader(ej))
    if err != nil {
        fmt.Printf("Inform sWorker to end seal error:%s\n", err)
    }
    if ! s || err != nil {
        dserv.RemoveMany(ctx, md.Cids(c))
    }
    md.Remove(c)
    return err
}

func SealBlock(ctx context.Context, c cid.Cid, r io.Reader, newBlock bool) ([]blocks.Block, error) {
    err := undefinedError()
    if err != nil {
        return nil, err
    }

    type sealParam struct {
        sk string
        newBlock bool `json:"new_block"`
    }

    kv := getKVFromSession(ctx)
    if kv == nil {
        return unexpectedError()
    }
    var bp = sealParam{
        sk: kv.sk,
        newBlock: newBlock,
    }
    bj, _ := json.Marshal(bp)

    resp, err := http.Post(sworkerBaseUrl + "/storage/seal?" + string(bj), "application/x-www-form-urlencoded", r)
    if err != nil {
		return nil, err
    }
    body, err := ioutil.ReadAll(resp.Body)
    resp.Body.Close()
    if err != nil {
		return nil, err
    }

    md := GetInstance()
    cidLen := len(c.String())

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
        if ! newBlock {
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
