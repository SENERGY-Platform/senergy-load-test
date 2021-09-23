package statistics

import (
	"context"
	"log"
	"sync"
	"time"
)

type Interface interface {
	EventProduce(duration time.Duration)
}

type Void struct{}

func (this Void) EventProduce(duration time.Duration) {}

func New(ctx context.Context, logAndResetInterval time.Duration) Interface {
	result := &Implementation{}
	result.Start(ctx, logAndResetInterval)
	return result
}

type Implementation struct {
	logAndResetInterval time.Duration
	producedEvents      []time.Duration
	eventMux            sync.Mutex
}

func (this *Implementation) EventProduce(duration time.Duration) {
	this.eventMux.Lock()
	defer this.eventMux.Unlock()
	this.producedEvents = append(this.producedEvents, duration)
}

func (this *Implementation) Start(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				this.log()
			}
		}
	}()

}

func (this *Implementation) log() {
	this.eventMux.Lock()
	defer this.eventMux.Unlock()

	avg, min, max := statistics(this.producedEvents)
	log.Println("LOG: produced events: \n\tthroughput:", len(this.producedEvents), "\n\tavg-latency:", avg.String(), "\n\tmin-latency:", min.String(), "\n\tmax-latency:", max.String())

	this.producedEvents = []time.Duration{}
}

func statistics(list []time.Duration) (avg time.Duration, min time.Duration, max time.Duration) {
	if len(list) > 0 {
		min = list[0]
	}
	sum := time.Duration(0)
	for _, element := range list {
		sum += element
		if min > element {
			min = element
		}
		if max < element {
			max = element
		}
	}
	avg = sum / time.Duration(len(list))
	return
}
