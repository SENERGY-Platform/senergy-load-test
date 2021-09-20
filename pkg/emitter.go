package pkg

import (
	"context"
	"math/rand"
	"time"
)

type Message struct {
	Device  string
	Service string
	Message string
}

func Emitter(ctx context.Context, out chan<- Message, device string, service string, interval time.Duration, message func() string) {
	go func() {
		//wait for random time between now and interval to offset emitter
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		if interval <= 1<<31-1 {
			time.Sleep(time.Duration(r.Int31n(int32(interval))))
		} else {
			time.Sleep(time.Duration(r.Int63n(int64(interval))))
		}
		emit(out, device, service, message)
		t := time.NewTicker(interval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				emit(out, device, service, message)
			}
		}
	}()
}

func emit(out chan<- Message, device string, service string, message func() string) {
	out <- Message{
		Device:  device,
		Service: service,
		Message: message(),
	}
}
