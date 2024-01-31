package course

import (
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

var TotalQuery int32

func Handler() {
	atomic.AddInt32(&TotalQuery, 1)
	time.Sleep(50 * time.Millisecond)
}

func CallHandler() {
	limiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 1) //每隔100ms生成一个令牌，最大QPS限制为10
	for {
		// ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		// defer cancel()
		// limiter.WaitN(ctx, 1) //阻塞，直到桶中有N个令牌。N=1时等价于Wait(ctx)
		// Handler()

		// if limiter.AllowN(time.Now(), 1) { //当前桶中是否至少还有N个令牌，如果有则返回true。N=1时等价于Allow(time.Time)
		// 	Handler()
		// }

		reserve := limiter.ReserveN(time.Now(), 1)
		time.Sleep(reserve.Delay()) //reserve.Delay()告诉你还需要等多久才会有充足的令牌，你就等吧
		Handler()
	}
}
