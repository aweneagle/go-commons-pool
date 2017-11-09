package pool

import (
	"errors"
	"sync/atomic"
	"time"
)

var ErrorPoolIsFull error = errors.New("pool is full")
var ErrorPoolIsEmpty error = errors.New("pool is empty")
var ErrorOptions error = errors.New("wrong options")
var ErrorTimeout error = errors.New("timeout")

type ObjFactory interface {
	// New, Destroy is Required, can't be nil
	New() (interface{}, error)
	Destroy(obj interface{}) error

	// Validate is Optional
	Validate(obj interface{}) error
}

type Options struct {
	// PoolSize > MaxIdelNum > MinIdelNum
	PoolSize   int32
	MaxIdelNum int32
	MinIdelNum int32

	Factory ObjFactory
}

type Pool struct {
	pool chan interface{}

	idelNum  int32
	totalNum int32
	options  Options
	serving  bool
}

func New(opt Options) *Pool {
	p := &Pool{
		pool:     make(chan interface{}, opt.PoolSize),
		idelNum:  0,
		totalNum: 0,
		options:  opt,
		serving:  false,
	}
	return p
}

/*
* Add() is not thread safe
 */
func (p *Pool) Add(num int32) error {
	if num > p.options.PoolSize-p.totalNum {
		return errors.New("too many object to add")
	}
	var i int32
	for i = 0; i < num; i++ {
		obj, err := p.options.Factory.New()
		if err != nil {
			return err
		}
		p.pool <- obj
		atomic.AddInt32(&p.idelNum, 1)
		atomic.AddInt32(&p.totalNum, 1)
	}
	return nil
}

func (p *Pool) Borrow() (interface{}, error) {
	if p.GetActivateNum() >= p.GetTotalNum() {
		return nil, ErrorPoolIsEmpty
	}
	select {
	case obj := <-p.pool:
		atomic.AddInt32(&p.idelNum, -1)
		return obj, nil

	case <-time.After(10 * time.Millisecond):
		return nil, ErrorTimeout
	}
}

func (p *Pool) Return(obj interface{}) {
	p.pool <- obj
	atomic.AddInt32(&p.idelNum, 1)
}

func (p *Pool) Destroy(obj interface{}) error {
	atomic.AddInt32(&p.totalNum, -1)
	return p.options.Factory.Destroy(obj)
}

func (p *Pool) Clean() {
	for toclean := p.totalNum; toclean > 0; toclean-- {
		if obj, _ := p.Borrow(); obj != nil {
			p.Destroy(obj)
		}
	}
}

func (p *Pool) GetTotalNum() int32 {
	return p.totalNum
}

func (p *Pool) GetIdelNum() int32 {
	return p.idelNum
}

func (p *Pool) GetActivateNum() int32 {
	return p.totalNum - p.idelNum
}

func (p *Pool) inc() error {
	var total int32 = atomic.AddInt32(&p.totalNum, 1)
	if total > p.options.PoolSize {
		atomic.AddInt32(&p.totalNum, -1)
		return ErrorPoolIsFull
	}
	return nil
}

func (p *Pool) dec() error {
	var total int32 = atomic.AddInt32(&p.totalNum, -1)
	if total < 0 {
		atomic.AddInt32(&p.totalNum, 1)
		return ErrorPoolIsEmpty
	}
	return nil
}

func (p *Pool) Serve() {
	if p.serving == true {
		return
	}
	p.serving = true
	//1. when idelNum < MinIdelNum, auto increase number of objs
	go func() {
		for {
			var opt = p.options
			if opt.MinIdelNum > p.idelNum {
				var need int32 = (opt.MinIdelNum+opt.MaxIdelNum)/2 - p.idelNum
				for ; need > 0; need-- {
					if err := p.inc(); err == nil {
						go func() {
							if obj, err := opt.Factory.New(); err != nil {
								p.dec()
							} else {
								p.Return(obj)
							}
						}()
					}
				}
			}
			//每秒检查一下是否需要更新对象池
			time.Sleep(10 * time.Millisecond)
		}
	}()
	//2. when idelNum > MaxIdelNum, auto decrease number of objs
	go func() {
		for {
			var opt = p.options
			var noneed int32 = p.idelNum - opt.MaxIdelNum
			if noneed > 0 {
				for ; noneed > 0; noneed-- {
					obj, _ := p.Borrow()
					if obj != nil {
						go p.Destroy(obj)
					}
				}
			}
			//每秒检查一下是否需要更新对象池
			time.Sleep(10 * time.Millisecond)
		}
	}()
}
