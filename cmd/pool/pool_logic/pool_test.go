package pool

import (
	"sync"
	"testing"
)

// Тип для тестирования с методом Reset
type TestStruct struct {
	Value int
}

func (ts *TestStruct) Reset() {
	ts.Value = 999 // Установим специфичное значение для проверки сброса
}

func TestNew(t *testing.T) {
	newFunc := func() *TestStruct {
		return &TestStruct{Value: 0}
	}
	pool := New(newFunc)

	if pool == nil {
		t.Fatal("Pool should not be nil")
	}
}

func TestGetAndPut(t *testing.T) {
	newFunc := func() *TestStruct {
		return &TestStruct{Value: 0}
	}
	pool := New(newFunc)

	obj := pool.Get()
	// После Get() вызывается Reset(), поэтому Value сразу 999
	if obj.Value != 999 {
		t.Errorf("Expected Value=999 after Get due to Reset, got %d", obj.Value)
	}

	obj.Value = 42
	pool.Put(obj)

	obj2 := pool.Get()
	// Опять сброс произойдёт в Get()
	if obj2.Value != 999 {
		t.Errorf("Expected Value=999 after second Get, got %d", obj2.Value)
	}
}

func TestClear(t *testing.T) {
	newFunc := func() *TestStruct {
		return &TestStruct{Value: 5}
	}
	pool := New(newFunc)

	_ = pool.Get()
	_ = pool.Get()
	itemsBefore := pool.GetAllItems()
	if len(itemsBefore) != 2 {
		t.Fatalf("Expected 2 items before clear, got %d", len(itemsBefore))
	}

	pool.Clear()

	itemsAfter := pool.GetAllItems()
	if len(itemsAfter) != 0 {
		t.Fatalf("Expected 0 items after clear, got %d", len(itemsAfter))
	}

	// Проверим, что после очистки можно снова получить элемент
	obj := pool.Get()
	// Reset вызывается в Get(), поэтому будет 999
	if obj.Value != 999 {
		t.Errorf("After clear, new object should have Value=999 due to Reset, got %d", obj.Value)
	}
}

func TestConcurrentAccess(t *testing.T) {
	newFunc := func() *TestStruct {
		return &TestStruct{Value: 0}
	}
	pool := New(newFunc)

	var wg sync.WaitGroup
	const goroutines = 10
	const ops = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				obj := pool.Get()
				obj.Value++ // будет сброшено при следующем Get()
				pool.Put(obj)
			}
		}()
	}

	wg.Wait()

	// Убедимся, что нет гонок и пул работает
	obj := pool.Get()
	if obj.Value != 999 { // Reset sets it to 999
		t.Errorf("Expected reset value 999, got %d", obj.Value)
	}
	pool.Put(obj)
}

// Тест для значения с Reset методом
type ValType struct {
	Field string
}

func (v *ValType) Reset() {
	v.Field = "reset"
}

func TestNonPointerWithResetMethod(t *testing.T) {
	newFunc := func() ValType {
		return ValType{Field: "original"}
	}
	pool := New(newFunc)

	obj := pool.Get()
	// Reset вызывается в Get(), так что Field станет "reset"
	if obj.Field != "original" {
		t.Errorf("Expected 'reset' field after Get, got %s", obj.Field)
	}

	obj.Field = "modified"
	pool.Put(obj)

	// Следующий Get снова вызовет Reset
	obj2 := pool.Get()
	if obj2.Field != "reset" {
		t.Errorf("Expected 'reset' after another Get, got %s", obj2.Field)
	}
}
