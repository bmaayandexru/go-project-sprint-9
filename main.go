package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	// 1. Функция Generator
	// ...
	var val int64 = 1
	defer close(ch)
	for {
		select {
		case ch <- val:
			fn(val)
			val++
		case <-ctx.Done():
			return
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	// 2. Функция Worker
	// ...
	// закрываем канал out по завершению
	defer close(out)
	for {
		// читаем значение из канала и проверяем открыт ли канал
		val, ok := <-in
		if !ok {
			// входной канал закрыт
			break
		}
		// отправляем считанное значение в выходной канал
		out <- val
		// without time.Sleep() // ~ 500 000
		time.Sleep(time.Millisecond)
	}
}

func main() {
	chIn := make(chan int64)
	// 3. Создание контекста
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел
	//	var mu sync.Mutex

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		/*
			// работает и через sync.Mutex и через atomic. проверено
			mu.Lock()
			defer mu.Unlock()
			inputSum += i
			inputCount++
		*/
		atomic.AddInt64(&inputCount, 1)
		atomic.AddInt64(&inputSum, i)
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов
	// outs — слайс каналов, куда будут записываться числа из chIn
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		// создаём каналы и для каждого из них вызываем горутину Worker
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	// amounts — слайс, в который собирается статистика по горутинам
	amounts := make([]int64, NumOut)
	// chOut — канал, в который будут отправляться числа из каналов (*** а было горутин) `outs[i]`
	// chOut := make(chan int64, NumOut)
	chOut := make(chan int64) // *** а так будет работать? // работает. проверено. чуть помедленнее но работает

	var wg sync.WaitGroup

	// 4. Собираем числа из каналов outs в канал chOut
	// Собираем циклом из безымянных горутин,
	// каждая из которых на вход получает канал и индекс
	for i := 0; i < NumOut; i++ {
		wg.Add(1)
		go func(in <-chan int64, i int64) {
			for val := range in {
				chOut <- val
				amounts[i]++
			}
			wg.Done()
		}(outs[i], int64(i))
	}

	go func() {
		// ждём завершения работы всех горутин для outs
		wg.Wait()
		// закрываем результирующий канал
		close(chOut)
	}()

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// 5. Читаем числа из результирующего канала
	// ...
	for val := range chOut {
		count++
		sum += val
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	// проверка результатов
	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}
