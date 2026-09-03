# Initialize the project directory and Go module
mkdir aggregator-engine && cd aggregator-engine
go mod init aggregator-engine

# Create the directory structure
mkdir -p cmd/api cmd/sync-worker \
  internal/adapter/schwab internal/adapter/ibkr internal/adapter/tradier \
  internal/domain internal/greeks internal/repository internal/service \
  internal/transport/http internal/transport/ws \
  pkg/crypto pkg/logger

# ---------------------------------------------------------
# Seed the runnable entrypoints
# ---------------------------------------------------------

cat <<EOF > cmd/api/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Starting API server on :8080...")
	
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK - API is running"))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
EOF

cat <<EOF > cmd/sync-worker/main.go
package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Starting background sync worker...")
	for {
		fmt.Println("Polling brokers for updates...")
		time.Sleep(5 * time.Second)
	}
}
EOF

# ---------------------------------------------------------
# Seed internal files with package declarations
# ---------------------------------------------------------

echo "package adapter" > internal/adapter/broker.go

for file in account position strategy trade; do 
  echo "package domain" > internal/domain/$file.go
done

echo "package greeks" > internal/greeks/evaluator.go

for file in postgres position_repo token_repo; do 
  echo "package repository" > internal/repository/$file.go
done

for file in auth_service portfolio_service strategy_service; do 
  echo "package service" > internal/service/$file.go
done

for file in router auth_handler portfolio_handler; do 
  echo "package http" > internal/transport/http/$file.go
done

echo "package ws" > internal/transport/ws/hub.go

# ---------------------------------------------------------
# Seed Docker Compose (PostgreSQL / TimescaleDB)
# ---------------------------------------------------------

cat <<EOF > docker-compose.yml
version: '3.8'
services:
  db:
    image: timescale/timescaledb:latest-pg15
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: portfolio
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
EOF

# Format all files
go fmt ./...