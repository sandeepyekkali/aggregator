package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/plaid/plaid-go/plaid"

	"aggregator-engine/internal/config"
	"aggregator-engine/internal/middleware"
	"aggregator-engine/internal/pkg/crypto"
	"aggregator-engine/internal/pkg/logger"
	"aggregator-engine/internal/repository"
	httptransport "aggregator-engine/internal/transport/http"
)

func main() {
	// 1. Load and Validate Configuration First
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Configuration failure", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 2. Initialize Logger
	log := logger.InitLogger(cfg.AppEnv)
	log.Info("Starting API server", slog.String("env", cfg.AppEnv))

	// 3. Connect to PostgreSQL using validated config
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Error("Failed to open database connection pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Error("Failed to ping PostgreSQL", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("Successfully established PostgreSQL connection")

	// 4. Initialize External API Clients (Plaid)
	plaidConfig := plaid.NewConfiguration()
	plaidConfig.AddDefaultHeader("PLAID-CLIENT-ID", cfg.PlaidClientID)
	plaidConfig.AddDefaultHeader("PLAID-SECRET", cfg.PlaidSecret)

	switch cfg.PlaidEnv {
	case "production":
		plaidConfig.UseEnvironment(plaid.Production)
	case "development":
		plaidConfig.UseEnvironment(plaid.Development)
	default:
		plaidConfig.UseEnvironment(plaid.Sandbox)
	}
	plaidClient := plaid.NewAPIClient(plaidConfig)
	log.Info("Plaid client initialized", slog.String("plaid_env", cfg.PlaidEnv))

	// 5. Initialize Repositories
	// Using the same 32-byte test key as the sync worker
	testKey := []byte("0123456789abcdef0123456789abcdef")
	encryptor, err := crypto.NewEncryptor(testKey)
	if err != nil {
		log.Error("Failed to initialize encryptor", slog.String("error", err.Error()))
		os.Exit(1)
	}

	plaidRepo := repository.NewPostgresPlaidRepo(db, encryptor)
	snapTradeRepo := repository.NewPostgresSnapTradeRepo(db, encryptor)
	positionRepo := repository.NewPostgresPositionRepo(db)
	txRepo := repository.NewPostgresTransactionRepo(db)
	userRepo := repository.NewPostgresUserRepo(db)

	// 6. Initialize HTTP Handlers
	portfolioHandler := httptransport.NewPortfolioHandler(positionRepo, userRepo)
	txHandler := httptransport.NewTransactionHandler(txRepo)
	userHandler := httptransport.NewUserHandler(userRepo)
	plaidHandler := httptransport.NewPlaidHandler(plaidClient, plaidRepo)
	snapTradeHandler := httptransport.NewSnapTradeHandler(cfg.SnapTradeClientID, cfg.SnapTradeConsumerKey, snapTradeRepo)

	// 7. Setup Router & Auth Middleware
	routerConfig := httptransport.RouterConfig{
		AuthMiddleware:   middleware.RequireAuth(cfg.SupabaseJWKSURL),
		PortfolioHandler: portfolioHandler,
		TxHandler:        txHandler,
		UserHandler:      userHandler,
		PlaidHandler:     plaidHandler,
		SnapTradeHandler: snapTradeHandler,
	}

	mux := httptransport.SetupRouter(routerConfig)

	// 8. Start Server
	srv := &http.Server{
		Addr: ":" + cfg.Port,
		// Pass the dynamic origin from your config into the global CORS middleware
		Handler:      middleware.CORS(cfg.FrontendOrigin)(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Info("HTTP server listening", slog.String("port", cfg.Port))
	if err := srv.ListenAndServe(); err != nil {
		log.Error("API server shut down unexpectedly", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
