package version

// Runtime is set at build time via -ldflags. Defaults to "gui".
const (
	RuntimeService = "service"
	RuntimeGUI     = "gui"
)

var Runtime = "gui"
