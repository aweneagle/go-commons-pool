package main

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

type ObjFactory interface {
	New() (interface{}, error)
	Destroy(obj interface{}) error
}

const (
	ErrorPoolIsFull error = errors.New("Pool Is Full")
)

type Pool struct {
	Factory ObjFactory
	OnError func(error)
	Size    int32
	MinIdle int32
	idel    int32
	total   int32

	//对象数量不足时，通知对象池需要增加对象了
	more chan int

	//对象需要销毁时，通知对象池删除
	less chan int

	//新增的对象
	new chan interface{}

	//对象池
	pool chan interface{}
}

func (p *pool) Serve() error {
	if p.Size < 0 {
		return errors.New(fmt.Sprintf("wrong pool size:%d", p.Size))
	}
	if p.MinIdle < 0 {
		return errors.New(fmt.Sprintf("wrong pool MinIdle:%d", p.MinIdle))
	}
	if p.Factory == nil {
		return errors.New("Pool.Facatory is missing")
	}
	if p.MinIdle == 0 {
		p.MinIdle = 32 //默认32
	}
	if p.Size == 0 {
		p.Size = 1024 //默认1024
	}
	p.more = make(chan int, p.Size)
	p.less = make(chan int, 1)
	p.pool = make(chan interface{}, p.Size)
	p.new = make(chan interface{}, p.Size)
	p.total, p.idel = 0

	//接到“新增对象”的通知
	go func() {
		for {
			num := <-p.more
			go func() {
				if err := p.Add(num); err != nil {
					p.OnError(err)
				}
			}()
		}
	}()

	//对象增加
	go func() {
		for {
			obj := <-p.new
			if p.total >= p.Size {
				go func() {
					err := p.Factory.Destroy(obj)
					if p.OnError != nil {
						p.OnError(err)
					}
				}()
				p.OnError(ErrorPoolIsFull)
			} else {
				p.total++
				atomic.AddInt32(&p.idle, 1)
				p.pool <- obj
			}
		}
	}()

	//对象销毁，更新total数
	go func() {
		for {
			less := <-p.less
			p.total -= less
		}
	}()

	//自动调节对象个数
	go func() {
		for {
			time.Sleep(10 * time.Second)
			todec := int(p.idle - p.MaxIdle)
			toinc := int(p.MinIdle - p.idle)
			//最多MaxIdle个空闲，超过的减去
			for i := 0; i < toinc; i++ {
				if err := p.Destroy(p.Borrow()); err != nil {
					p.OnError(err)
				}
			}
			//最少MinIdle个空闲，不够的补上
			for i := 0; i < todec; i++ {
				if err := p.Add(1); err != nil {
					p.OnError(err)
				}
			}
		}
	}()

	return nil
}

func (p *Pool) Destroy(obj interface{}) error {
	p.less <- 1
	return p.Factory.Destroy(obj)
}

func (p *Pool) Add(num int) error {
	for i := 0; i < num; i++ {
		obj, err := p.Factory.New()
		if err != nil {
			return err
		}
		p.new <- obj
	}
	return nil
}

func (p *Pool) Borrow() interface{} {
	atomic.AddInt32(&p.idle, -1)
	if p.idle < 0 {
		p.more <- 1
	}
	return <-p.pool
}

func (p *Pool) Return(obj interface{}) {
	atomic.AddInt32(&p.idle, 1)
	p.pool <- obj
}
