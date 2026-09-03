package domain

// RiskLevel (Final Spec §13).
//
// The numeric risk_score is backend-internal; the frontend receives only
// the level. Risk is defined only for OPEN portals.
type RiskLevel string

const (
	RiskLow      RiskLevel = "LOW"      // 0..25
	RiskMedium   RiskLevel = "MEDIUM"   // >25..50
	RiskHigh     RiskLevel = "HIGH"     // >50..75
	RiskCritical RiskLevel = "CRITICAL" // >75..100
)
