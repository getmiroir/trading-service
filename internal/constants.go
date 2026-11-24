package internal

const (
	AppDirName            = ".miroir"
	ProgramDataDir        = `C:\ProgramData`
	ProgramDataAppDirName = "Miroir"
)

// BinanceReferralURL is set at build time via -ldflags. Defaults to a placeholder.
var BinanceReferralURL = "https://accounts.binance.com/en/register"

// BybitReferralURL is set at build time via -ldflags. Defaults to a placeholder.
var BybitReferralURL = "https://www.bybit.com/en/tr-register"
