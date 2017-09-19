package pool

import (
	"errors"
	"sync/atomic"
	"time"
)

var ErrorPoolIsFull error = errors.New("pool is full")
var ErrorPoolIsEmpty error = errors.New("pool is empty")
var ErrorOptions error = errors.New("wrong options")

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

type pool struct {
	pool chan interface{}

	idelNum  int32
	totalNum int32

	options Options
}

func New(opt Options) *pool {
	p := &pool{
		pool:     make(chan interface{}, opt.PoolSize),
		idelNum:  0,
		totalNum: 0,
		options:  opt,
	}
	p.serve()
	return p
}

func (p *pool) Borrow() (interface{}, error) {
	obj := <-p.pool
	atomic.AddInt32(&p.idelNum, -1)
	return obj, nil
}

func (p *pool) Return(obj interface{}) {
	p.pool <- obj
	atomic.AddInt32(&p.idelNum, 1)
}

func (p *pool) Destroy(obj interface{}) error {
	atomic.AddInt32(&p.totalNum, -1)
	return p.options.Destroy(obj)
}

func (p *pool) GetTotalNum() int32 {
	return p.totalNum
}

func (p *pool) GetIdelNum() int32 {
	return p.idelNum
}

func (p *pool) inc() error {
	var total int32 = atomic.AddInt32(&p.totalNum, 1)
	if total > p.options.PoolSize {
		atomic.AddInt32(&p.totalNum, -1)
		return ErrorPoolIsFull
	}
	return nil
}

func (p *pool) dec() error {
	var total int32 = atomic.AddInt32(&p.totalNum, -1)
	if total < 0 {
		atomic.AddInt32(&p.totalNum, 1)
		return ErrorPoolIsEmpty
	}
	return nil
}

func (p *pool) serve() {
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
								for {
									//p.desc() must be succefully done, because we already "consume" a pit of pool by p.inc()
									if fail2desc := p.dec(); fail2desc == nil {
										break
									}
								}
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
