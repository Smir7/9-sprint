package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммыb
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {

	var i int64 = 1
	for {
		defer close(ch)
		select {
		case <-ctx.Done():
			return
		case ch <- i:
			fn(i)
			i++
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	defer close(out)
	for {
		v, ok := <-in
		if !ok {
			break
		}
		out <- v
		time.Sleep(1 * time.Millisecond)
	}
}

func main() {
	chIn := make(chan int64)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел

	go Generator(ctx, chIn, func(i int64) {
		var mu sync.Mutex
		mu.Lock()
		defer mu.Unlock()
		inputSum += i
		inputCount++
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов

	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	amounts := make([]int64, NumOut)
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup
	for i := 0; i < NumOut; i++ {
		wg.Add(1)
		go func(in <-chan int64, num int64) {
			defer wg.Done()
			for v := range in {
				chOut <- v
				amounts[num]++
			}
		}(outs[i], int64(i))
	}

	go func() {
		wg.Wait()
		close(chOut)
	}()

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	for {
		v, ok := <-chOut
		if !ok {
			break
		}
		count++
		sum += v
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
