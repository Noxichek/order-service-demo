// package main говорит Go, что этот файл — точка входа для создания исполняемой программы.
// Каждое приложение на Go должно начинаться с пакета main.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-service/internal/cache"
	"order-service/internal/config"
	"order-service/internal/handler"
	"order-service/internal/kafka"
	"order-service/internal/model"
	"order-service/internal/store"
)

func main() {
	// ------------------------------------------------------------
	// Шаг 1. Загружаем настройки из переменных окружения.
	cfg := config.Load()

	// ------------------------------------------------------------
	// Шаг 2. Создаём корневой контекст с возможностью отмены.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ------------------------------------------------------------
	// Шаг 3. Подключаемся к базе данных PostgreSQL.
	st, err := store.NewStore(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer st.Close()

	// ------------------------------------------------------------
	// Шаг 4. Создаём кэш с ограничением размера и TTL.
	// Первый аргумент — максимальное количество записей (10000),
	// второй — время жизни записи (10 минут).
	c := cache.New(10000, 10*time.Minute)

	// Прогреваем кэш из БД, чтобы при старте быстро отвечать на запросы.
	if err := c.LoadFromDB(func() ([]*model.Order, error) {
		return st.GetAllOrders(ctx)
	}); err != nil {
		log.Printf("warning: failed to load cache from DB: %v", err)
	}

	// ------------------------------------------------------------
	// Шаг 5. Запускаем Kafka consumer.
	// Он будет читать сообщения из топика, сохранять заказы в БД и обновлять кэш.
	consumer := kafka.NewConsumer(cfg.KafkaBroker, cfg.KafkaTopic, "order-service-group", st, c)
	consumer.Start(ctx)
	defer consumer.Close()

	// ------------------------------------------------------------
	// Шаг 6. Настраиваем HTTP-сервер.
	h := handler.New(st, c)

	mux := http.NewServeMux()
	mux.HandleFunc("/order/", h.GetOrder)

	srv := &http.Server{
		Addr:              cfg.BindAddress + ":" + cfg.HTTPPort, // слушаем только на указанном адресе (по умолч. 127.0.0.1)
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// ------------------------------------------------------------
	// Шаг 7. Graceful shutdown: ловим сигналы ОС и аккуратно останавливаем сервис.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")

		cancel() // останавливаем Kafka consumer

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
	}()

	// ------------------------------------------------------------
	// Шаг 8. Запускаем HTTP-сервер.
	log.Printf("HTTP server listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
	log.Println("server stopped")
}
