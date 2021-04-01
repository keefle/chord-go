package main

import (
	"testing"
)

func NewMockCaller(succ, succspred string) *MockCaller {
	return &MockCaller{succ: succ, succspred: succspred}
}

type MockCaller struct {
	calls     [][]string
	succ      string
	succspred string
}

func (mc *MockCaller) Call(addr, proc string, args interface{}, reply interface{}) error {
	mc.calls = append(mc.calls, []string{addr, proc})
	switch proc {
	case "Lookup":
		lr := reply.(*LookupResp)
		lr.Addr = "localhost:8085"
	case "GetPred":
		gpr := reply.(*GetPredResp)
		gpr.Addr = "localhost:8070"
	case "SetSucc":
		// ssr := reply.(*SetSuccResp)
		// ssr.Addr = "localhost:8085"
	case "SetPred":
		// spr := reply.(*SetPredResp)
		// spr.Addr = "localhost:8085"
	}
	return nil
}

func TestNodeJoin(t *testing.T) {
	self := "localhost:8080"
	introducer := "localhost:8081"
	succ := "localhost:8085"
	succspred := "localhost:8070"

	n := NewNode(self)
	mc := NewMockCaller(succ, succspred)

	n.join(introducer, mc)

	if n.Successor != succ {
		t.Errorf("successor is wrong. Expected (%v), found (%v)", succ, n.Successor)
	}

	if n.Predecessor != succspred {
		t.Errorf("successor is wrong. Expected (%v), found (%v)", succspred, n.Predecessor)
	}
}
