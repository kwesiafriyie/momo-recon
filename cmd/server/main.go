package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kwesiafriyie/momo-recon/internal/config"
	"github.com/kwesiafriyie/momo-recon/internal/handler"
	"github.com/kwesiafriyie/momo-recon/internal/repository"
	"github.com/kwesiafriyie/momo-recon/internal/service"
	"github.com/kwesiafriyie/momo-recon/internal/worker"
	"github.com/kwesiafriyie/momo-recon/pkg/momo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// --- Database ---
	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: connect: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("db: ping: %v", err)
	}
	log.Println("db: connected")

	// --- Repositories ---
	invoiceRepo := repository.NewInvoiceRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	eventRepo := repository.NewEventRepository(db)

	// --- MoMo Client ---
	momoClient := momo.NewClient(
		cfg.MoMoBaseURL,
		cfg.MoMoSubscriptionKey,
		cfg.MoMoAPIUserID,
		cfg.MoMoAPIKey,
		cfg.MoMoCallbackURL,
		cfg.MoMoTargetEnv,
	)

	// --- Services ---
	invoiceSvc := service.NewInvoiceService(invoiceRepo)
	reconcilerSvc := service.NewReconciliationService(invoiceRepo, transactionRepo)
	momoSvc := service.NewMoMoService(momoClient, invoiceRepo, transactionRepo, eventRepo)

	// --- Workers ---
	eventWorker := worker.NewEventWorker(eventRepo, transactionRepo, reconcilerSvc)
	pollingWorker := worker.NewPollingWorker(momoClient, transactionRepo, reconcilerSvc)

	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	go eventWorker.Run(workerCtx)
	go pollingWorker.Run(workerCtx)

	// --- Handlers & Routes ---
	invoiceHandler := handler.NewInvoiceHandler(invoiceSvc)
	momoHandler := handler.NewMoMoHandler(momoSvc)

	mux := http.NewServeMux()

	// Invoices
	mux.HandleFunc("POST /api/invoices", invoiceHandler.Create)
	mux.HandleFunc("GET /api/invoices", invoiceHandler.List)
	mux.HandleFunc("GET /api/invoices/{code}", invoiceHandler.Get)

	// Payments & Transactions
	mux.HandleFunc("POST /api/pay", momoHandler.Pay)
	mux.HandleFunc("GET /api/transactions", momoHandler.ListTransactions)

	// MoMo Webhook
	mux.HandleFunc("POST /api/momo/callback", momoHandler.Callback)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// --- HTTP Server ---
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server: listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-quit
	log.Println("server: shutting down...")
	cancelWorkers()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server: forced shutdown: %v", err)
	}
	log.Println("server: stopped")
}
