package main

import (
	"fmt"

	pool "github.com/shatrunoff/yap_metrics/cmd/pool/pool_logic"
)

// Пример структуры с методом Reset
type ExampleStruct struct {
	ID    int
	Name  string
	Data  []byte
	Cache map[string]int
}

// generate:reset
func (es *ExampleStruct) Reset() {
	if es == nil {
		return
	}

	es.ID = 0
	es.Name = ""
	es.Data = es.Data[:0]
	clear(es.Cache)
}

func main() {
	// Создаем пул для ExampleStruct
	pool := pool.New(func() ExampleStruct {
		return ExampleStruct{
			Data:  make([]byte, 0, 1024),
			Cache: make(map[string]int),
		}
	})

	// Получаем объект из пула
	obj1 := pool.Get()
	obj1.ID = 1
	obj1.Name = "test"
	obj1.Data = append(obj1.Data, 1, 2, 3)
	obj1.Cache["key"] = 100

	fmt.Printf("Объект после использования: ID=%d, Name=%s, Data len=%d\n",
		obj1.ID, obj1.Name, len(obj1.Data))

	// Возвращаем объект в пул
	pool.Put(obj1)

	// Получаем объект снова - он будет сброшен
	obj2 := pool.Get()
	fmt.Printf("Объект после сброса: ID=%d, Name=%s, Data len=%d, Cache len=%d\n",
		obj2.ID, obj2.Name, len(obj2.Data), len(obj2.Cache))

	// Получаем статистику
	allItems := pool.GetAllItems()
	fmt.Printf("Всего создано объектов: %d\n", len(allItems))

	// Очищаем пул
	pool.Clear()
}
