package clients

// CopyTarget represents a copy trading target configuration
type CopyTarget struct {
	Address           *string        `json:"address"`
	Enabled           bool           `json:"enabled"`
	ExposureFactor    float64        `json:"exposureFactor"`
	MinimumEquity     *float64       `json:"minimumEquity"`
	MaximumLeverage   *int           `json:"maximumLeverage"`
	LeverageOverrides map[string]int `json:"leverageOverrides"`
	TpslSafetyEnabled *bool          `json:"tpslSafetyEnabled"`
	ExcludedSymbols   []string       `json:"excludedSymbols"`
}

// IsTpslSafetyEnabled reports whether TP/SL placement is enabled for this target.
// A nil value (older backend) is treated as enabled.
func (ct *CopyTarget) IsTpslSafetyEnabled() bool {
	return ct.TpslSafetyEnabled == nil || *ct.TpslSafetyEnabled
}

// AddressShort returns a shortened version of the address for display
func (ct *CopyTarget) AddressShort() string {
	if ct.Address == nil || *ct.Address == "" {
		return "<none>"
	}
	addr := *ct.Address
	if len(addr) <= 10 {
		return addr
	}
	return addr[:6] + "..." + addr[len(addr)-4:]
}

// DefaultCopyTarget returns a CopyTarget with default values
func DefaultCopyTarget() CopyTarget {
	return CopyTarget{
		Address:           nil,
		Enabled:           true,
		ExposureFactor:    1.0,
		MinimumEquity:     nil,
		MaximumLeverage:   nil,
		LeverageOverrides: make(map[string]int),
	}
}

// CopyTargets represents the collection of copy targets
type CopyTargets struct {
	CopyEnabled      bool                  `json:"copyEnabled"`
	ByAccount        map[string]CopyTarget `json:"byAccount"`
	ArchivedAccounts []string              `json:"archivedAccounts"` // Accounts the copy-service must not sync
}

// DefaultCopyTargets returns CopyTargets with default values
func DefaultCopyTargets() CopyTargets {
	return CopyTargets{
		CopyEnabled:      false,
		ByAccount:        make(map[string]CopyTarget),
		ArchivedAccounts: nil,
	}
}

// CopyServiceReportResponse represents the response from the copy-service-report endpoint
type CopyServiceReportResponse struct {
	Success bool   `json:"success"`
	Cause   string `json:"cause"`
}
