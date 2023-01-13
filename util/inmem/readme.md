# inmem

* 带失效时间的LRU Cache
* 同时支持协程安全和非协程安全两种方式
* 根据facebook开源库修改而成

# interface

        type Cache interface {
                Add(key, value interface{}, expiresAt time.Time)
                Get(key interface{}) (interface{}, bool)
                Remove(key interface{})
                Len() int
        }

# example

        import (
                "code.byted.org/gopkg/inmem"
                "fmt"
                "time"
        )
        
        func main() {
                lru := inmem.NewUnlocked(10)
                lru.Add(1, 1, time.Now().Add(5*time.Minute))
                fmt.Println(lru.Get(1))
                lru.Remove(1)
                fmt.Println(lru.Len())
        }
