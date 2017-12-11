# go-commons-pool

使用示例：

1. 加载库

```golang
import (
    pool "github.com/aweneagle/go-commons-pool"
)
```

2. 定义对象生成方法
```golang
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
```

3. 使用对象池

* Borrow() 借用对象
* Return() 返还对象
* Destroy() 销毁对象（例如连接失效的情况下）

```golang
func main() {
  p := &pool.Pool{
		Size:    1000,   //设置对象池大小
		MaxIdle: 30,     //设置最大Idle数目
		MinIdle: 10,     //设置最小Idle数目
		Factory: Fact{}, //ObjFactory 对象工厂， 需实现 ObjFactory interface
  }
  p.Serve()
  o := p.Borrow() //获取对象
  //do something with object 
  if ok {
     p.Return(o) 
  } elese {
     p.Destroy(o)
  }
}
```
