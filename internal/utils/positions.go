package utils

// SideToMultiplier converts a side string ("Buy" or "Sell") to a multiplier (1 or -1)
func SideToMultiplier(side string) int {
	if side == "Buy" {
		return 1
	}
	return -1
}

// QuantityToSide converts a quantity (positive or negative) to a side string ("Buy" or "Sell")
func QuantityToSide(quantity float64) string {
	if quantity >= 0 {
		return "Buy"
	}
	return "Sell"
}
