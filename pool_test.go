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
		t.Fatalf("%s, pool status: total=%d, idel=%d, activate=%d", msg, p.GetTotalNum(), p.GetIdelNum(), p.GetActivateNum())
	}
}

func create() (interface{}, error) {
	total := atomic.AddInt32(&sum, 1)
	obj := &TestObj{total}
	return obj, nil
}

func destroy(obj interface{}) error {
	return nil
}

func TestPoolMain(t *testing.T) {
	sum = 0
	p, _ := New(Options{
		PoolSize:   1000,
		MaxIdelNum: 30,
		MinIdelNum: 10,
		New:        create,
		Destroy:    destroy,
	})
	time.Sleep(15 * time.Millisecond)
	n, err := p.Borrow()
	assert(err == nil, "something wrong when borrow", t, p)
	assert(n.(*TestObj).num == 1, "the first num should be 1", t, p)
	assert(p.GetTotalNum() == 20, "the total num should be 20", t, p)
	assert(p.GetIdelNum() == 19, "the total num should be 19", t, p)
	assert(p.GetActivateNum() == 1, "the total num should be 1", t, p)

	p.Destroy(n)
	assert(p.GetTotalNum() == 19, "the total num should be 19", t, p)
	assert(p.GetIdelNum() == 19, "the total num should be 19", t, p)
	assert(p.GetActivateNum() == 0, "the total num should be 0", t, p)

	n, err = p.Borrow()
	assert(p.GetTotalNum() == 19, "the total num should be 19", t, p)
	assert(p.GetIdelNum() == 18, "the total num should be 18", t, p)
	assert(p.GetActivateNum() == 1, "the total num should be 1", t, p)
}

/**
*   test pool auto increase/decrease ability based on min pool size of 3
 */
func TestPoolMinSize(t *testing.T) {
	sum = 0
	p, _ := New(Options{
		PoolSize:   3,
		MaxIdelNum: 2,
		MinIdelNum: 1,
		New:        create,
		Destroy:    destroy,
	})

	time.Sleep(15 * time.Millisecond)
	assert(p.GetTotalNum() == 1, "n: the total num should be 1", t, p)
	assert(p.GetIdelNum() == 1, "n: the total num should be 1", t, p)
	assert(p.GetActivateNum() == 0, "n: the total num should be 0", t, p)

	n1, _ := p.Borrow()
	assert(p.GetTotalNum() == 1, "n1: the total num should be 1", t, p)
	assert(p.GetIdelNum() == 0, "n1: the idel num should be 0", t, p)
	time.Sleep(15 * time.Millisecond)
	assert(p.GetTotalNum() == 2, "n1: the total num should be 2", t, p)
	assert(p.GetIdelNum() == 1, "n1: the idel num should be 1", t, p)

	n2, _ := p.Borrow()
	time.Sleep(15 * time.Millisecond)
	assert(p.GetTotalNum() == 3, "n2: the total num should be 2", t, p)
	assert(p.GetIdelNum() == 1, "n2: the idel num should be 2", t, p)

	n3, _ := p.Borrow()
	time.Sleep(15 * time.Millisecond)
	assert(p.GetTotalNum() == 3, "n3: the total num should be 3", t, p)
	assert(p.GetIdelNum() == 0, "n3: the idel num should be 0", t, p)

	n4, err := p.Borrow()
	assert(err == ErrorPoolIsFull, "n4: the pool should be full", t, p)
	assert(n4 == nil, "n4: obj should not be able to borrowed", t, p)
	assert(p.GetTotalNum() == 3, "n4: the total num should be 3", t, p)
	assert(p.GetIdelNum() == 0, "n4: the idel num should be 0", t, p)

	p.Return(n1)
	p.Return(n2)
	p.Return(n3)
	assert(p.GetTotalNum() == 3, "nn: the total num should be 3", t, p)
	assert(p.GetIdelNum() == 3, "nn: the idel num should be 3", t, p)
	time.Sleep(15 * time.Millisecond)
	assert(p.GetTotalNum() == 2, "nm: the total num should be 1", t, p)
	assert(p.GetIdelNum() == 2, "nm: the idel num should be 1", t, p)

}
