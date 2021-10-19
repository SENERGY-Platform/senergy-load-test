package pkg

import (
	"context"
	"math/rand"
	"time"
)

type Message struct {
	Info    map[string]string
	Message string
}

func Emitter(ctx context.Context, out chan<- Message, info map[string]string, interval time.Duration, message func() string) {
	if interval > 0 {
		go func() {
			//wait for random time between now and interval to offset emitter
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			if interval <= 1<<31-1 {
				time.Sleep(time.Duration(r.Int31n(int32(interval))))
			} else {
				time.Sleep(time.Duration(r.Int63n(int64(interval))))
			}
			emit(out, info, message)
			t := time.NewTicker(interval)
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					emit(out, info, message)
				}
			}
		}()
	}
}

func emit(out chan<- Message, info map[string]string, message func() string) {
	out <- Message{
		Info:    info,
		Message: message(),
	}
}
