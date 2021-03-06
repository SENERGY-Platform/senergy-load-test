package statistics

import (
	"context"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Interface interface {
	EventProduce(duration time.Duration)
	EventEmitted()
	CommandsHandled()
}

type Void struct{}

func (this Void) EventProduce(duration time.Duration) {}
func (this Void) EventEmitted()                       {}
func (this Void) CommandsHandled()                    {}

func New(ctx context.Context, logAndResetInterval time.Duration) Interface {
	result := &Implementation{}
	result.Start(ctx, logAndResetInterval)
	return result
}

type Implementation struct {
	logAndResetInterval  time.Duration
	producedEvents       []time.Duration
	emittedCount         uint64
	commandsHandledCount uint64
	eventMux             sync.Mutex
}

func (this *Implementation) EventProduce(duration time.Duration) {
	this.eventMux.Lock()
	defer this.eventMux.Unlock()
	this.producedEvents = append(this.producedEvents, duration)
}

func (this *Implementation) EventEmitted() {
	atomic.AddUint64(&this.emittedCount, 1)
}

func (this *Implementation) CommandsHandled() {
	atomic.AddUint64(&this.commandsHandledCount, 1)
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

	produced := len(this.producedEvents)
	emitted := atomic.LoadUint64(&this.emittedCount)
	commands := atomic.LoadUint64(&this.commandsHandledCount)

	median, avg, min, max := statistics(this.producedEvents)
	log.Println("LOG: produced events:", "\n\tcommands:", commands, "\n\temitted:", emitted, "\n\tproduced:", produced, "\n\tmedian-produce-time:", median.String(), "\n\tavg-produce-time:", avg.String(), "\n\tmin-produce-tim:", min.String(), "\n\tmax-produce-tim:", max.String())

	this.producedEvents = []time.Duration{}
	atomic.StoreUint64(&this.emittedCount, 0)
	atomic.StoreUint64(&this.commandsHandledCount, 0)
}

func statistics(list []time.Duration) (median time.Duration, avg time.Duration, min time.Duration, max time.Duration) {
	sort.Slice(list, func(i, j int) bool {
		return list[i] < list[j]
	})
	count := len(list)
	if len(list) > 0 {
		min = list[0]
		max = list[count-1]
	}

	if count > 2 {
		if count%2 == 0 {
			median = list[count/2]
		} else {
			median = (list[count/2] + list[(count/2)-1]) / 2
		}
	}

	sum := time.Duration(0)
	for _, element := range list {
		sum += element
	}
	if len(list) > 0 {
		avg = sum / time.Duration(len(list))
	}
	return
}
