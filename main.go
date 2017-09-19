package main

import (
	"./pool"
	"fmt"
	"sync/atomic"
	"time"
)

var sum int32

type TestObj struct {
	num int32
}

func main() {
	p := pool.New(pool.Options{
		PoolSize:   1000,
		MaxIdelNum: 30,
		MinIdelNum: 10,
		New: func() (interface{}, error) {
			total := atomic.AddInt32(&sum, 1)
			obj := &TestObj{total}
			fmt.Println("object:", obj.num, " create")
			return obj, nil
		},
		Destroy: func(obj interface{}) error {
			fmt.Println("object:", obj.(*TestObj).num, " destroied")
			return nil
		},
	})
	var test = func(x int64) {
		c, _ := p.Borrow()
		fmt.Println(x, "] object:", c.(*TestObj).num, " borrow out, total:", p.GetTotalNum(), ",idel:", p.GetIdelNum())
		time.Sleep(50 * time.Millisecond)
		if x%10 == 0 {
			//p.Destroy(c)
			p.Return(c)
		} else {
			p.Return(c)
		}
	}

	var i int64 = 0
	//for ; i < 10; i++ {
	for {
		go test(i)
		i++
		time.Sleep(1 * time.Millisecond)
	}
	test(i)
}
