package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/plaid/plaid-go/plaid"

	"aggregator-engine/internal/adapter"
	"aggregator-engine/internal/config"
	"aggregator-engine/internal/middleware"
	"aggregator-engine/internal/pkg/crypto"
	"aggregator-engine/internal/pkg/logger"
	"aggregator-engine/internal/repository"
	"aggregator-engine/internal/service"
	"aggregator-engine/internal/snaptrade"
	httptransport "aggregator-engine/internal/transport/http"
)

func main() {
	// 1. Load and Validate Configuration First
	cfg, err := config.Load()
	if err != nil {
		// We use standard log here because our structured logger hasn't initialized yet
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

	// 4. Initialize External API Clients (Plaid & SnapTrade)
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

	// New SnapTrade Client
	snapClient := snaptrade.NewClient(cfg.SnapTradeClientID, cfg.SnapTradeConsumerKey)
	log.Info("SnapTrade client initialized")

	// 5. Initialize Repositories & Services

	// Initialize the encryptor using the same 32-byte test key as the sync worker
	testKey := []byte("0123456789abcdef0123456789abcdef")
	encryptor, err := crypto.NewEncryptor(testKey)
	if err != nil {
		log.Error("Failed to initialize encryptor", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Inject the encryptor into our isolated token repositories
	plaidRepo := repository.NewPostgresPlaidRepo(db, encryptor)
	snapTradeRepo := repository.NewPostgresSnapTradeRepo(db, encryptor)

	positionRepo := repository.NewPostgresPositionRepo(db)
	txRepo := repository.NewPostgresTransactionRepo(db)
	userRepo := repository.NewPostgresUserRepo(db)

	factory := adapter.NewAdapterFactory()
	portfolioService := service.NewPortfolioService(factory, positionRepo)

	// 6. Initialize HTTP Handlers
	portfolioHandler := httptransport.NewPortfolioHandler(portfolioService)
	txHandler := httptransport.NewTransactionHandler(txRepo)
	userHandler := httptransport.NewUserHandler(userRepo)

	plaidHandler := httptransport.NewPlaidHandler(plaidClient, plaidRepo)
	snapTradeHandler := httptransport.NewSnapTradeHandler(snapClient, snapTradeRepo)

	// 7. Setup Router & Auth Middleware
	// We now pull the validated JWKS URL directly from our strict Config
	authMiddleware := middleware.RequireAuth(cfg.SupabaseJWKSURL)

	mux := http.NewServeMux()

	// --- Public Routes ---
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "healthy"}`))
	})
	// Open route for new signups
	mux.HandleFunc("POST /api/v1/users", userHandler.HandleCreateUser)

	// --- Protected Routes ---

	// Basic Tier Features (Inherited by Pro & Premium)
	mux.Handle("GET /api/v1/portfolio", authMiddleware(middleware.RequireMinimumTier("basic")(http.HandlerFunc(portfolioHandler.HandleGetPortfolio))))
	mux.Handle("GET /api/v1/transactions", authMiddleware(middleware.RequireMinimumTier("basic")(http.HandlerFunc(txHandler.HandleGetTransactions))))

	mux.Handle("POST /api/v1/plaid/create-link-token", authMiddleware(middleware.RequireMinimumTier("basic")(http.HandlerFunc(plaidHandler.HandleCreateLinkToken))))
	mux.Handle("POST /api/v1/plaid/exchange-public-token", authMiddleware(middleware.RequireMinimumTier("basic")(http.HandlerFunc(plaidHandler.HandleExchangePublicToken))))

	// Premium Tier Features (Strictly Gated)
	mux.Handle("POST /api/v1/snaptrade/link", authMiddleware(middleware.RequireMinimumTier("basic")(http.HandlerFunc(snapTradeHandler.HandleCreateLinkToken))))

	// 8. Start Server
	srv := &http.Server{
		Addr: ":" + cfg.Port,
		// Pass the dynamic origin from your config into the middleware
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
