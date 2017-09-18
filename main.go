package main

import (
	"./pool"
	"fmt"
	"time"
)

var sum int

type TestObj struct {
	num int
}

func main() {
	go func() {
		sum++
		time.Sleep(1 * time.Second)
	}()
	p := pool.New(pool.Options{
		PoolSize:   1000,
		MaxIdelNum: 30,
		MinIdelNum: 10,
		New: func() (interface{}, error) {
			obj := &TestObj{sum}
			var num int = 0
			var dev int = 0
			fmt.Println("object:", obj.num, " create", num/dev)
			return obj, nil
		},
		Destroy: func(obj interface{}) error {
			fmt.Println("object:", obj.(*TestObj).num, " destroied")
			return nil
		},
	})
	p.Serve()
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
