package cache

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"test-task/order-service/internal/domain"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type Cache interface {
	Add(key string, value interface{}) bool
	Get(key string) interface{}
}

type Item struct {
	Key   string
	Value interface{}
}

type LRUCache struct {
	capacity int
	queue    *list.List
	mutex    *sync.RWMutex
	items    map[string]*list.Element
}

func New(cap int) *LRUCache {
	return &LRUCache{
		capacity: cap,
		queue:    list.New(),
		mutex:    new(sync.RWMutex),
		items:    make(map[string]*list.Element),
	}
}

func (c *LRUCache) Add(key string, value interface{}) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if item, exists := c.items[key]; exists {
		c.queue.MoveToFront(item)
		item.Value.(*Item).Value = value
		return true
	}

	if c.queue.Len() == c.capacity {
		c.clear()
	}

	item := &Item{
		Key:   key,
		Value: value,
	}

	element := c.queue.PushFront(item)
	c.items[item.Key] = element

	return true
}

func (c *LRUCache) Get(key string) interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	element, exists := c.items[key]
	if !exists {
		return nil
	}

	c.queue.MoveToFront(element)
	return element.Value.(*Item).Value
}

func (c *LRUCache) Remove(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if value, found := c.items[key]; found {
		c.deleteItem(value)
	}

	return true
}

func (c *LRUCache) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.items)
}

func (c *LRUCache) clear() {
	if element := c.queue.Back(); element != nil {
		c.deleteItem(element)
	}
}

func (c *LRUCache) deleteItem(elem *list.Element) {
	item := c.queue.Remove(elem).(*Item)
	delete(c.items, item.Key)
}

func (c *LRUCache) EvacuateToDB(log *log.Logger, dbUri string) error {
	const op = "cache.EvacuateToDB"

	if c.Len() == 0 {
		log.Print("Cache is already clear")
		return nil
	}

	const createCacheTable = `
		CREATE TABLE IF NOT EXISTS cache (
			id CHAR(19) PRIMARY KEY,
			data JSONB NOT NULL,
			UNIQUE (id, data)
		);
	`

	db, err := sqlx.Open("pgx", dbUri)
	if err != nil {
		return fmt.Errorf("%s: open db connection: %w", op, err)
	}

	_, err = db.Exec(createCacheTable)
	if err != nil {
		return fmt.Errorf("%s: creating table: %w", op, err)
	}

	q := `INSERT INTO cache (id, data) VALUES ($1, $2)`

	stmt, err := db.Prepare(q)
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	count := 0
	for elem := c.queue.Front(); elem != nil; elem = elem.Next() {
		order := elem.Value.(*Item).Value.(*domain.Order)

		if _, err := stmt.Exec(order.OrderUid, order); err != nil {
			return fmt.Errorf("%s: saving entry: %w", op, err)
		}

		count++
		log.Printf("Cache entry with id: [%s] has been saved", order.OrderUid)
	}
	log.Printf("cache loop ended, queue len: [%d]", count)

	return nil
}

func (c *LRUCache) RestoreFromDB(log *log.Logger, ctx context.Context, dbUri string) error {
	const op = "cache.RestoreFromDB"

	const qCheckIfExists = `SELECT 
		    COUNT(table_name)
		FROM 
		    information_schema.tables 
		WHERE 
		    table_schema LIKE 'public' AND 
		    table_type LIKE 'BASE TABLE' AND
			table_name = 'cache';
	`

	db, err := sqlx.Open("pgx", dbUri)
	if err != nil {
		return fmt.Errorf("%s: open db connection: %w", op, err)
	}

	stmt, err := db.Prepare(qCheckIfExists)
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var count int

	err = stmt.QueryRowContext(ctx).Scan(&count)
	if err != nil {
		return fmt.Errorf("%s: checking if cache table exists: %w", op, err)
	}

	if count == 0 {
		log.Printf("Cache is empty")
		return nil
	}

	stmt, err = db.PrepareContext(ctx, "SELECT id, data FROM cache")
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return fmt.Errorf("%s: querying stmt: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return fmt.Errorf("%s: scanning cache rows: %w", op, err)
		}

		var order domain.Order
		err = json.Unmarshal(data, &order)

		if err != nil {
			return fmt.Errorf("%s: unmarshalling data: %w", op, err)
		}

		c.Add(id, &order)
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("%s: scanning rows: %w", op, err)
	}

	_, err = db.ExecContext(ctx, "TRUNCATE TABLE cache")
	if err != nil {
		return fmt.Errorf("%s: truncating cache table: %w", op, err)
	}

	log.Printf("Cache fully restored")
	return nil
}
