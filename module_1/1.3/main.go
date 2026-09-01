package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// worker - функция, выполняемая в отдельной горутине.
// Она читает числа из канала jobs и выводит их в stdout.
func worker(id int, jobs <-chan int, wg *sync.WaitGroup) {
	defer wg.Done() // по завершении работы уменьшаем счетчик WaitGroup
	for num := range jobs {
		fmt.Printf("Воркер %d: %d\n", id, num)
		// Небольшая задержка для наглядности (можно убрать)
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	// Проверяем, что передан аргумент с количеством воркеров
	if len(os.Args) < 2 {
		fmt.Println("Использование: go run main.go <количество_воркеров>")
		os.Exit(1)
	}

	// Преобразуем аргумент в целое число
	numWorkers, err := strconv.Atoi(os.Args[1])
	if err != nil || numWorkers <= 0 {
		fmt.Println("Ошибка: количество воркеров должно быть положительным целым числом")
		os.Exit(1)
	}

	// Создаем контекст с отменой для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Канал для передачи заданий
	jobs := make(chan int)

	// WaitGroup для ожидания завершения всех воркеров
	var wg sync.WaitGroup

	// Запускаем N воркеров
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, jobs, &wg)
	}

	// Главная горутина будет генерировать данные и отправлять в канал
	go func() {
		counter := 0
		for {
			select {
			case <-ctx.Done():
				// При получении сигнала отмены закрываем канал и выходим из генератора
				close(jobs)
				return
			default:
				counter++
				jobs <- counter
			}
		}
	}()

	// Обработка сигналов ОС для корректного завершения (Ctrl+C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("Запущено %d воркеров. Нажмите Ctrl+C для остановки.\n", numWorkers)

	// Блокируемся до получения сигнала
	<-sigCh
	fmt.Println("\nПолучен сигнал остановки, завершаем работу...")

	// Отменяем контекст, чтобы генератор прекратил отправку и закрыл канал
	cancel()

	// Ждем, пока все воркеры завершат обработку оставшихся заданий
	wg.Wait()
	fmt.Println("Все воркеры остановлены. Программа завершена.")
}