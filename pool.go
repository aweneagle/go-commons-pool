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

type Options struct {
	// PoolSize > MaxIdelNum > MinIdelNum
	PoolSize   int32
	MaxIdelNum int32
	MinIdelNum int32

	// New, Destroy is Required, can't be nil
	New     func() (obj interface{}, err error)
	Destroy func(obj interface{}) (err error)

	// Validate is Optional
	Validate func(obj interface{}) (err error)
}

type Pool struct {
	pool chan interface{}

	idelNum  int32
	totalNum int32

	options Options
}

func New(opt Options) *Pool {
	p := &Pool{
		pool:     make(chan interface{}, opt.PoolSize),
		idelNum:  0,
		totalNum: 0,
		options:  opt,
	}
	p.serve()
	return p
}

func (p *Pool) Borrow() (interface{}, error) {
	if p.GetActivateNum() >= p.GetTotalNum() {
		return nil, ErrorPoolIsFull
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
	return p.options.Destroy(obj)
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

func (p *Pool) serve() {
	//1. when idelNum < MinIdelNum, auto increase number of objs
	go func() {
		for {
			var opt = p.options
			if opt.MinIdelNum > p.idelNum {
				var need int32 = (opt.MinIdelNum+opt.MaxIdelNum)/2 - p.idelNum
				for ; need > 0; need-- {
					if err := p.inc(); err == nil {
						go func() {
							if obj, err := opt.New(); err != nil {
								p.dec()
							} else {
								p.Return(obj)
							}
						}()
					}
				}
			}
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
			time.Sleep(10 * time.Millisecond)
		}
	}()
}
