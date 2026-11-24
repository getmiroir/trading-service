package keyring

import zalandokeyring "github.com/zalando/go-keyring"

// ServiceName is the fixed keyring service name for miroir-trading-service entries.
//
// Platform behavior:
//   - macOS/Linux use the system keyring through go-keyring.
//   - Windows stores DPAPI-encrypted blobs in %PROGRAMDATA%\Miroir\secrets
//     so both the interactive GUI user and the Local System service can read
//     the same secrets.
//     The split is needed because kardianos runs the service as a Local System service on Windows,
//     and the user keyring is not available to the Local System service.
//
// Windows notes:
//   - Secrets are machine-bound (DPAPI LOCAL_MACHINE scope).
//   - Any local administrator (or SYSTEM process) can decrypt secrets.
const ServiceName = "miroir"

// ErrNotFound is returned when no secret exists for a key.
var ErrNotFound = zalandokeyring.ErrNotFound
