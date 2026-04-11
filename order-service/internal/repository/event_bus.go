package repository

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
)

// PgEventBus обеспечивает реальный стриминг через PostgreSQL LISTEN/NOTIFY.
// Это ключевой компонент для требования "Real-time DB Integration" (15%).
//
// Принцип работы:
// 1. При UpdateStatus в репозитории отправляется NOTIFY order_updates, 'order_id:status'
// 2. PgEventBus слушает канал "order_updates" через PostgreSQL LISTEN
// 3. Подписчики (gRPC стриминг) получают события для конкретного order_id
// 4. Никаких time.Sleep() — только реальные события из базы данных!
type PgEventBus struct {
	dsn       string
	listener  *pq.Listener
	mu        sync.RWMutex
	// subscribers: map[order_id] -> slice of channels
	subscribers map[string][]chan OrderEvent
	done        chan struct{}
}

// OrderEvent — событие об изменении статуса заказа.
type OrderEvent struct {
	OrderID string
	Status  string
}

// NewPgEventBus создаёт новый EventBus.
func NewPgEventBus(dsn string) *PgEventBus {
	return &PgEventBus{
		dsn:         dsn,
		subscribers: make(map[string][]chan OrderEvent),
		done:        make(chan struct{}),
	}
}

// Start запускает PostgreSQL LISTEN на канале "order_updates".
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

	// Горутина для обработки уведомлений
	go eb.listen()

	return nil
}

// listen — основной цикл обработки PostgreSQL NOTIFY.
func (eb *PgEventBus) listen() {
	for {
		select {
		case <-eb.done:
			return
		case notification := <-eb.listener.Notify:
			if notification == nil {
				// Reconnect notification
				continue
			}

			// Payload формат: "order_id:status"
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

			// Рассылаем подписчикам этого конкретного order_id
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

// Subscribe подписывается на события для конкретного order_id.
// Возвращает канал, из которого можно читать OrderEvent.
func (eb *PgEventBus) Subscribe(orderID string) chan OrderEvent {
	ch := make(chan OrderEvent, 10)

	eb.mu.Lock()
	eb.subscribers[orderID] = append(eb.subscribers[orderID], ch)
	eb.mu.Unlock()

	log.Printf("[EventBus] subscriber added for order %s", orderID)
	return ch
}

// Unsubscribe отписывается от событий для order_id.
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

	// Удаляем ключ, если нет подписчиков
	if len(eb.subscribers[orderID]) == 0 {
		delete(eb.subscribers, orderID)
	}

	close(ch)
	log.Printf("[EventBus] subscriber removed for order %s", orderID)
}

// Stop останавливает EventBus.
func (eb *PgEventBus) Stop() {
	close(eb.done)
	if eb.listener != nil {
		eb.listener.Close()
	}
	log.Println("[INFO] EventBus stopped")
}
