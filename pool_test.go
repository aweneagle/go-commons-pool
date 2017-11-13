package pool

import (
	//"fmt"
	"sync/atomic"
	"testing"
	"time"
)

var sum int32

type TestObj struct {
	num int32
}

func assert(assert bool, msg string, t *testing.T, p *Pool) {
	if assert {
		//fmt.Sprintf("test success:%s", msg)
	} else {
		t.Fatalf("%s, pool status: total=%d, idel=%d, activate=%d", msg, p.GetTotalNum(), p.GetIdleNum(), p.GetActivateNum())
	}
}

type Fact struct {
}

func (f Fact) New() (interface{}, error) {
	total := atomic.AddInt32(&sum, 1)
	obj := &TestObj{total}
	return obj, nil
}

func (f Fact) Destroy(obj interface{}) error {
	return nil
}

func TestPoolMain(t *testing.T) {
	sum = 0
	p := &Pool{
		Size:    1000,
		MaxIdle: 30,
		MinIdle: 10,
		Factory: Fact{},
	}
	p.Serve()
	n := p.Borrow()
	assert(n.(*TestObj).num == 1, "the first num should be 1", t, p)
	assert(p.GetTotalNum() == 1, "the total num should be 1", t, p)
	assert(p.GetIdleNum() == 0, "the idle num should be 0", t, p)
	assert(p.GetActivateNum() == 1, "the active num should be 1", t, p)

	p.Destroy(n)
	time.Sleep(10 * time.Millisecond)
	assert(p.GetTotalNum() == 0, "the total num should be 0", t, p)
	assert(p.GetIdleNum() == 0, "the idle num should be 0", t, p)
	assert(p.GetActivateNum() == 0, "the active num should be 0", t, p)

	p.Add(3)
	time.Sleep(10 * time.Millisecond)
	assert(p.GetTotalNum() == 3, "the total num should be 3", t, p)
	assert(p.GetIdleNum() == 3, "the idle num should be 3", t, p)
	assert(p.GetActivateNum() == 0, "the active num should be 0", t, p)
}

/**
*   test pool auto increase/decrease ability based on min pool size of 3
 */
func TestPoolMinSize(t *testing.T) {
	sum = 0
	p := &Pool{
		Size:    10,
		MaxIdle: 6,
		MinIdle: 4,
		Factory: Fact{},
	}
	p.Serve()
	//测试自动增加功能
	p.autoIncDec()
	time.Sleep(10 * time.Millisecond)
	assert(p.GetTotalNum() == 4, "n1: the total num should be 4", t, p)
	assert(p.GetIdleNum() == 4, "n1: the total num should be 4", t, p)
	//测试自动减少功能
	p.Add(8)
	p.autoIncDec()
	time.Sleep(10 * time.Millisecond)
	assert(p.GetIdleNum() == 9, "n1: the total num should be 9", t, p)
	p.autoIncDec()
	time.Sleep(10 * time.Millisecond)
	assert(p.GetIdleNum() == 8, "n1: the total num should be 8", t, p)

}
