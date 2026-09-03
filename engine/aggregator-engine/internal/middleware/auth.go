package middleware

import (
	"context"
	"net/http"
	"strings"

	"log/slog"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

func RequireAuth(jwksURL string) func(http.Handler) http.Handler {
	// Fetch Supabase's public keys once and cache them in memory
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{})
	if err != nil {
		slog.Error("Failed to create JWKS from URL", slog.String("error", err.Error()))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}

			// Extract token from "Bearer <token>"
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Parse and mathematically verify the JWT using the public JWKS
			token, err := jwt.Parse(tokenString, jwks.Keyfunc)
			if err != nil || !token.Valid {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// 1. Extract the User ID
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}
			userID := claims["sub"].(string)

			// 2. Extract the Tier (Default to "basic" if not found)
			tier := "basic"
			if appMetadata, ok := claims["app_metadata"].(map[string]interface{}); ok {
				if t, exists := appMetadata["tier"]; exists {
					tier = t.(string)
				}
			}

			// 3. Inject BOTH into the request context
			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "user_tier", tier)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
