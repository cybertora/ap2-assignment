package repository

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
)

type PgEventBus struct {
	dsn      string
	listener *pq.Listener
	mu       sync.RWMutex
	// subscribers: map[order_id] -> slice of channels
	subscribers map[string][]chan OrderEvent
	done        chan struct{}
}

type OrderEvent struct {
	OrderID string
	Status  string
}

func NewPgEventBus(dsn string) *PgEventBus {
	return &PgEventBus{
		dsn:         dsn,
		subscribers: make(map[string][]chan OrderEvent),
		done:        make(chan struct{}),
	}
}

func (eb *PgEventBus) Start() error {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("[EventBus WARN] listener event: %v", err)
		}
	}

	eb.listener = pq.NewListener(eb.dsn, 10*time.Second, time.Minute, reportProblem)

	if err := eb.listener.Listen("order_updates"); err != nil {
		return err
	}

	log.Println("[INFO] EventBus: LISTEN on channel 'order_updates' started")

	go eb.listen()

	return nil
}

func (eb *PgEventBus) listen() {
	for {
		select {
		case <-eb.done:
			return
		case notification := <-eb.listener.Notify:
			if notification == nil {
				continue
			}

			parts := strings.SplitN(notification.Extra, ":", 2)
			if len(parts) != 2 {
				log.Printf("[EventBus WARN] invalid payload: %s", notification.Extra)
				continue
			}

			event := OrderEvent{
				OrderID: parts[0],
				Status:  parts[1],
			}

			log.Printf("[EventBus] received: order_id=%s status=%s", event.OrderID, event.Status)

			eb.mu.RLock()
			subs, ok := eb.subscribers[event.OrderID]
			if ok {
				for _, ch := range subs {
					select {
					case ch <- event:
					default:
						// Канал заполнен — пропускаем (не блокируем)
						log.Printf("[EventBus WARN] subscriber channel full for order %s", event.OrderID)
					}
				}
			}
			eb.mu.RUnlock()
		}
	}
}

func (eb *PgEventBus) Subscribe(orderID string) chan OrderEvent {
	ch := make(chan OrderEvent, 10)

	eb.mu.Lock()
	eb.subscribers[orderID] = append(eb.subscribers[orderID], ch)
	eb.mu.Unlock()

	log.Printf("[EventBus] subscriber added for order %s", orderID)
	return ch
}

func (eb *PgEventBus) Unsubscribe(orderID string, ch chan OrderEvent) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subs := eb.subscribers[orderID]
	for i, sub := range subs {
		if sub == ch {
			eb.subscribers[orderID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	if len(eb.subscribers[orderID]) == 0 {
		delete(eb.subscribers, orderID)
	}

	close(ch)
	log.Printf("[EventBus] subscriber removed for order %s", orderID)
}

func (eb *PgEventBus) Stop() {
	close(eb.done)
	if eb.listener != nil {
		eb.listener.Close()
	}
	log.Println("[INFO] EventBus stopped")
}
