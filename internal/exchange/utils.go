package exchange

import (
	"fmt"
	"strings"
)

const (
	expectedPositionMode    = "one-way"
	tradingSettingsTutorial = " — see Margin and Position Mode in the tutorial panel"
)

func formatAccountValidationError(violations []string) error {
	return fmt.Errorf("account validation failed:\n  - %s", strings.Join(violations, "\n  - "))
}
