package middleware

import (
	"net/http"
)

// TierWeights defines the numeric hierarchy of your application's tiers.
// Spacing them by 10 allows you to easily slip new tiers in later (e.g., a "Pro-Plus" at 25)
// without rewriting your existing route requirements.
var TierWeights = map[string]int{
	"basic":   10, // Plaid data, EOD updates
	"pro":     20, // Snapshot options, advanced analytics
	"premium": 30, // Real-time SnapTrade options, IBKR routing
	"elite":   40, // Custom webhooks, algorithmic API access
}

// RequireMinimumTier wraps an HTTP handler and blocks users who do not meet the minimum weight.
func RequireMinimumTier(requiredTier string) func(http.Handler) http.Handler {
	requiredWeight, valid := TierWeights[requiredTier]
	if !valid {
		// Defensive programming: fail closed if a typo is made in the route setup
		requiredWeight = 999
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userTier, ok := r.Context().Value("user_tier").(string)
			if !ok || userTier == "" {
				userTier = "basic"
			}

			userWeight := TierWeights[userTier]

			if userWeight < requiredWeight {
				http.Error(w, "Forbidden: Insufficient tier level for this feature", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
