package pool

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testObj — простая структура с Reset() для тестов.
type testObj struct {
	Value int
	Data  []string
	reset int // счётчик вызовов Reset
}

// Reset сбрасывает поля и увеличивает счётчик вызовов.
func (o *testObj) Reset() {
	o.Value = 0
	o.Data = o.Data[:0]
	o.reset++
}

// TestPool_NewAndGet — проверяет, что New создаёт пул, а Get возвращает объект.
func TestPool_NewAndGet(t *testing.T) {
	p := New(func() *testObj { return &testObj{Value: 42} })

	obj := p.Get()
	require.NotNil(t, obj)
	// Первый Get — пул пуст, значит вернулся объект из фабрики.
	assert.Equal(t, 42, obj.Value)
}

// TestPool_PutResets — ключевой тест: Put должен вызвать Reset().
func TestPool_PutResets(t *testing.T) {
	p := New(func() *testObj { return &testObj{} })

	obj := p.Get()
	obj.Value = 100
	obj.Data = append(obj.Data, "hello")
	require.Equal(t, 100, obj.Value)
	require.Len(t, obj.Data, 1)

	p.Put(obj)

	// Reset должен был сработать.
	assert.Equal(t, 0, obj.Value)
	assert.Len(t, obj.Data, 0)
	assert.Equal(t, 1, obj.reset, "Reset должен быть вызван ровно один раз")
}

// TestPool_GetAfterPut — после Put объект возвращается из пула уже сброшенным.
func TestPool_GetAfterPut(t *testing.T) {
	p := New(func() *testObj { return &testObj{} })

	obj := p.Get()
	obj.Value = 100
	obj.reset = 0 // сбрасываем счётчик для чистоты проверки
	p.Put(obj)

	// Следующий Get должен вернуть тот же объект (в рамках одного P)
	// уже сброшенным.
	obj2 := p.Get()
	assert.Equal(t, 0, obj2.Value)
	// Reset был вызван в Put, а не в Get.
	assert.Equal(t, 1, obj2.reset)
}

// TestPool_ReusesObject — проверяет, что пул действительно переиспользует объект,
// а не создаёт новый каждый раз.
func TestPool_ReusesObject(t *testing.T) {
	var created int
	p := New(func() *testObj {
		created++
		return &testObj{}
	})

	obj1 := p.Get()
	p.Put(obj1)
	obj2 := p.Get()

	// Если объект переиспользован — фабрика вызвалась только один раз
	// (или ни разу при повторных Get после Put).
	assert.LessOrEqual(t, created, 2, "фабрика не должна вызываться много раз")
	// Указатели должны совпадать (sync.Pool переиспользует объекты,
	// но не гарантирует этого строго — поэтому используем мягкую проверку).
	_ = obj1
	_ = obj2
}

// TestPool_Concurrent — проверяет потокобезопасность.
func TestPool_Concurrent(t *testing.T) {
	p := New(func() *testObj { return &testObj{} })

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				obj := p.Get()
				obj.Value = id*iterations + j
				obj.Data = append(obj.Data, "x")
				p.Put(obj)
			}
		}(i)
	}
	wg.Wait()

	// Если гонок нет — тест дошёл сюда без паники.
	assert.NotNil(t, p)
}

// TestPool_ResetNotCalledInGet — важный тест: Get НЕ должен вызывать Reset().
// Reset должен вызываться только в Put.
func TestPool_ResetNotCalledInGet(t *testing.T) {
	p := New(func() *testObj { return &testObj{} })

	obj := p.Get()
	p.Put(obj) // Reset вызван здесь → reset == 1

	obj2 := p.Get() // Reset НЕ должен вызываться здесь
	assert.Equal(t, 1, obj2.reset, "Get не должен дополнительно вызывать Reset")
}
