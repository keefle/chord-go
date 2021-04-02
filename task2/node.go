package main

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/rpc"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type Caller interface {
	Call(addr, proc string, args interface{}, reply interface{}) error
}

type Node struct {
	Addr        string
	Successor   string
	Predecessor string

	mufile    sync.Mutex
	fileTable map[uint64]string

	mufing      sync.Mutex
	fingerTable []string
}

func NewNode(addr string) *Node {
	return &Node{Addr: addr, Successor: addr, Predecessor: addr,
		fingerTable: make([]string, 20, 20),
		fileTable:   make(map[uint64]string)}
}

type LookupResp struct {
	Addr string
	ID   uint64
}

type UploadFileReq struct {
	Content  []byte
	Filename string
	ID       uint64
}

type UploadFileResp struct {
	Err error
}

type ShareFilesReq struct {
	PredID uint64
	ID     uint64
	Addr   string
}

type ShareFilesResp struct {
	Err error
}

type RetrieveFileReq struct {
	Filename string
	ID       uint64
}

type RetrieveFileResp struct {
	Content  []byte
	Filename string
	ID       uint64
	Err      error
}

type GetPredResp struct {
	Addr string
	ID   uint64
}

type GetSuccResp struct {
	Addr string
	ID   uint64
}

type SetPredResp struct {
	Addr string
	ID   uint64
}

type SetSuccResp struct {
	Addr string
	ID   uint64
}

func (n *Node) leave(client Caller) {
	n.mufile.Lock()
	defer n.mufile.Unlock()

	var err error

	var ssr SetSuccResp
	err = client.Call(n.Predecessor, "SetSucc", n.Successor, &ssr)
	if err != nil {
		log.Println(err)
	}

	var spr SetPredResp
	err = client.Call(n.Successor, "SetPred", n.Predecessor, &spr)
	if err != nil {
		log.Println(err)
	}

	id := n.id()
	for fileid, filename := range n.fileTable {
		// go func(filename string, id uint64, client Caller) {
		buff, err := ioutil.ReadFile(filepath.Join(strconv.FormatUint(id, 10), filename))
		if err != nil {
			log.Println(err)
			continue
		}

		client.Call(n.Successor, "UploadFile", UploadFileReq{
			Filename: filename,
			Content:  buff,
			ID:       fileid,
		}, &UploadFileResp{})

		if err := os.Remove(filepath.Join(strconv.FormatUint(n.id(), 10), filename)); err != nil {
			log.Println(err)
			return
		}

		delete(n.fileTable, fileid)
		// }(filename, n.id(), client)
	}

	if err := os.RemoveAll(strconv.FormatUint(n.id(), 10)); err != nil {
		log.Println(err)
		return
	}

	var empty string
	client.Call(n.Predecessor, "Stabilize", n.Predecessor, &empty)
	log.Print("sending stabalize call")
}

func (n *Node) join(peeraddr string, client Caller) {

	log.Printf("joining through peer (%v)", peeraddr)

	var lr LookupResp
	var err error

	err = client.Call(peeraddr, "Lookup", n.id(), &lr)
	if err != nil {
		log.Println(err)
	}

	n.setSucc(lr.Addr)
	log.Printf("setting successor to (%v) of ID (%v)", lr.Addr, ID(lr.Addr))

	var gpr GetPredResp
	err = client.Call(n.Successor, "GetPred", "", &gpr)
	if err != nil {
		log.Println("GetPred", err)
	}

	n.setPred(gpr.Addr)
	log.Printf("setting predecessor to (%v) of ID (%v)", gpr.Addr, ID(gpr.Addr))

	var spr SetPredResp
	err = client.Call(n.Successor, "SetPred", n.Addr, &spr)
	if err != nil {
		log.Println(err)
	}

	log.Printf("setting self (%v) of ID (%v) as predecessor of successor (%v) of ID (%v)", n.Addr, ID(n.Addr), n.Successor, ID(n.Successor))

	var ssr SetSuccResp
	err = client.Call(n.Predecessor, "SetSucc", n.Addr, &ssr)
	if err != nil {
		log.Println(err)
	}

	log.Printf("setting self (%v) of ID (%v) as successor of predecessor (%v) of ID (%v)", n.Addr, ID(n.Addr), n.Predecessor, ID(n.Predecessor))

	n.calcFingerTable()

	log.Print("calculated self's finger table")

	var empty string
	err = client.Call(n.Predecessor, "CalcFingerTable", empty, &empty)
	if err != nil {
		log.Println(err)
	}
	log.Print("calculated predecessor's finger table")

	err = client.Call(n.Successor, "CalcFingerTable", empty, &empty)
	if err != nil {
		log.Println(err)
	}
	log.Print("calculated successor's finger table")

	err = client.Call(n.Successor, "ShareFiles", ShareFilesReq{PredID: ID(n.Predecessor), ID: n.id(), Addr: n.Addr}, &ShareFilesResp{})
	if err != nil {
		log.Println(err)
	}

	client.Call(n.Predecessor, "Stabilize", n.Addr, &empty)
	if err != nil {
		log.Println(err)
	}
	log.Print("sending stabalize call")

	// Somehow Get Related Files from successor
}

func (n *Node) calcFingerTable() {
	for i := 0; i < len(n.fingerTable); i++ {
		lr := n.lookupbasic((n.id()+uint64(math.Pow(2, float64(i))))%1048576, NewRPCCaller())
		n.fingerTable[i] = lr.Addr
	}
}

func (n *Node) id() uint64 {
	hashBytes := sha1.Sum([]byte(n.Addr))
	return binary.BigEndian.Uint64(hashBytes[:]) % 1048576
}

func ID(addr string) uint64 {
	hashBytes := sha1.Sum([]byte(addr))
	return binary.BigEndian.Uint64(hashBytes[:]) % 1048576
}

func (n *Node) lookup(id uint64, caller Caller) LookupResp {

	if n.id() == ID(n.Successor) {
		return LookupResp{Addr: n.Successor, ID: ID(n.Successor)}
	}

	p := n.Addr
	ft := n.fingerTable[0]

	switch {
	case ID(p) < id && id <= ID(ft):
		return LookupResp{Addr: ft, ID: ID(ft)}
	case ID(p) > ID(ft) && ID(p) < id:
		return LookupResp{Addr: ft, ID: ID(ft)}
	case ID(p) > ID(ft) && id <= ID(ft):
		return LookupResp{Addr: ft, ID: ID(ft)}
	}

	p = ft

	for i := 1; i < len(n.fingerTable); i++ {
		ft = n.fingerTable[i]
		switch {
		case ID(p) < id && id <= ID(ft):
			var lr LookupResp
			caller.Call(p, "Lookup", id, &lr)
			return lr
		case ID(p) > ID(ft) && ID(p) < id:
			var lr LookupResp
			caller.Call(p, "Lookup", id, &lr)
			return lr
		case ID(p) > ID(ft) && id <= ID(ft):
			var lr LookupResp
			caller.Call(p, "Lookup", id, &lr)
			return lr
		}

		p = ft
	}

	var lr LookupResp
	caller.Call(p, "Lookup", id, &lr)
	return lr
}

func (n *Node) lookupbasic(id uint64, caller Caller) LookupResp {
	switch {
	case n.id() == ID(n.Successor):
		return LookupResp{Addr: n.Successor, ID: ID(n.Successor)}
	case n.id() < id && id <= ID(n.Successor):
		return LookupResp{Addr: n.Successor, ID: ID(n.Successor)}
	case n.id() > ID(n.Successor) && n.id() < id:
		return LookupResp{Addr: n.Successor, ID: ID(n.Successor)}
	case n.id() > ID(n.Successor) && id <= ID(n.Successor):
		return LookupResp{Addr: n.Successor, ID: ID(n.Successor)}
	default:
		var lr LookupResp
		caller.Call(n.Successor, "Lookup", id, &lr)
		return lr
	}
}

func (n *Node) setPred(predecessor string) {
	n.Predecessor = predecessor
}

func (n *Node) setSucc(successor string) {
	n.Successor = successor
}

func (n *Node) getPred() string {
	return n.Predecessor
}

func (n *Node) getSucc() string {
	return n.Successor
}

func (n *Node) stabilize(origin string, caller Caller) error {
	n.calcFingerTable()
	log.Print("Stabalized: DONE")
	if n.Predecessor == origin {
		return nil
	}

	go func(origin string, caller Caller) {
		var empty string
		if err := caller.Call(n.Predecessor, "Stabilize", origin, &empty); err != nil {
			log.Println("Error in stabilize: ", err)
		}
	}(origin, caller)

	return nil
}

func (n *Node) retrieveFile(rf RetrieveFileReq) (*RetrieveFileResp, error) {
	n.mufile.Lock()
	defer n.mufile.Unlock()

	content, err := ioutil.ReadFile(filepath.Join(strconv.FormatUint(n.id(), 10), rf.Filename))
	if err != nil {
		return nil, err
	}

	return &RetrieveFileResp{Filename: rf.Filename, Content: content, ID: ID(rf.Filename)}, nil
}

func (n *Node) RetrieveFile(rf RetrieveFileReq, rfr *RetrieveFileResp) error {
	rfrr, err := n.retrieveFile(rf)
	if err != nil {
		rfr.Err = err
		return err
	}

	rfr.Content = rfrr.Content
	rfr.Filename = rfrr.Filename
	rfr.ID = rfrr.ID

	return nil
}

func (n *Node) uploadFile(uf UploadFileReq) error {
	n.mufile.Lock()
	defer n.mufile.Unlock()

	if err := os.MkdirAll(strconv.FormatUint(n.id(), 10), 0700); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(strconv.FormatUint(n.id(), 10), uf.Filename), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	if _, err := f.Write(uf.Content); err != nil {
		return err
	}

	n.fileTable[uf.ID] = uf.Filename

	return nil
}

func (n *Node) shareFiles(sf ShareFilesReq, client Caller) error {
	n.mufile.Lock()
	defer n.mufile.Unlock()

	id := n.id()
	for fileid, filename := range n.fileTable {

		switch {
		case sf.PredID < fileid && fileid <= sf.ID:
		case sf.PredID > sf.ID && sf.PredID < fileid:
		case sf.PredID > sf.ID && fileid <= sf.ID:
		default:
			continue
		}

		buff, err := ioutil.ReadFile(filepath.Join(strconv.FormatUint(id, 10), filename))
		if err != nil {
			log.Println(err)
			continue
		}

		client.Call(sf.Addr, "UploadFile", UploadFileReq{
			Filename: filename,
			Content:  buff,
			ID:       fileid,
		}, &UploadFileResp{})

		if err := os.Remove(filepath.Join(strconv.FormatUint(n.id(), 10), filename)); err != nil {
			log.Println(err)
			continue
		}

		delete(n.fileTable, fileid)
	}

	return nil
}

func (n *Node) ShareFiles(sf ShareFilesReq, sfr *ShareFilesResp) error {
	if err := n.shareFiles(sf, NewRPCCaller()); err != nil {
		sfr.Err = err
		return err
	}

	return nil
}

func (n *Node) UploadFile(uf UploadFileReq, ufr *UploadFileResp) error {
	if err := n.uploadFile(uf); err != nil {
		ufr.Err = err
		return err
	}

	return nil
}

func (n *Node) Stabilize(origin string, empty *string) error {
	return n.stabilize(origin, NewRPCCaller())
}

func (n *Node) Lookup(id uint64, lr *LookupResp) error {
	llr := n.lookup(id, NewRPCCaller())

	lr.Addr = llr.Addr
	lr.ID = llr.ID
	return nil
}

func (n *Node) SetSucc(succ string, ssr *SetSuccResp) error {
	n.setSucc(succ)

	ssr.Addr = succ
	ssr.ID = ID(succ)

	return nil
}

func (n *Node) SetPred(pred string, spr *SetPredResp) error {
	n.setPred(pred)

	spr.Addr = pred
	spr.ID = ID(pred)

	return nil
}

func (n *Node) GetSucc(empty string, gsr *GetSuccResp) error {
	addr := n.getSucc()

	gsr.Addr = addr
	gsr.ID = ID(addr)

	return nil
}

func (n *Node) GetPred(empty string, gpr *GetPredResp) error {
	addr := n.getPred()

	gpr.Addr = addr
	gpr.ID = ID(addr)

	return nil
}

func (n *Node) CalcFingerTable(empty string, emptyreply *string) error {
	n.calcFingerTable()
	return nil
}

func (n *Node) printFingerTable() {
	fmt.Println("i  | address        | ID")
	for i := 0; i < len(n.fingerTable); i++ {
		fmt.Printf("%02d (%7d) | %v | %7d\n", i, (n.id()+uint64(math.Pow(2, float64(i))))%1048576, n.fingerTable[i], ID(n.fingerTable[i]))
	}
}

func (n *Node) printFileTable() {
	fmt.Println("Key     | Filename")
	for fileid, filename := range n.fileTable {
		fmt.Printf("(%7d) | %v\n", fileid, filename)
	}
}

type RPCCaller struct{}

func (rc *RPCCaller) Call(addr, proc string, args interface{}, reply interface{}) error {
	conn, err := rpc.Dial("tcp4", addr)
	if err != nil {
		return err
	}

	if err := conn.Call("Node."+proc, args, reply); err != nil {
		return err
	}

	if err := conn.Close(); err != nil {
		return err
	}

	return nil
}

func NewRPCCaller() *RPCCaller {
	return &RPCCaller{}
}
