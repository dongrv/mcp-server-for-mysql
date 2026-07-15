package execution

import (
	"crypto/subtle"
	"fmt"

	"github.com/dongrv/mcp-server-for-mysql/internal/config"
	"github.com/dongrv/mcp-server-for-mysql/internal/sqlguard"
)

// Authorize applies quick or strict policy to a complete canonical intent.
// No hash is retained: callers must present the same hash with the same intent
// on the execution request. It has no identity or replay protection; that
// remains an upstream gateway responsibility.
func Authorize(mode config.Mode, intent Intent, confirmation Confirmation) (Decision, Preview, error) {
	if mode != config.QuickMode && mode != config.StrictMode {
		return Decision{State: PolicyDenied}, invalidPreview(intent), fmt.Errorf("invalid execution mode %q", mode)
	}

	preview, err := BuildPreview(intent)
	if err != nil {
		return Decision{State: PolicyDenied}, preview, err
	}
	return decidePreview(mode, preview, confirmation), preview, nil
}

func decidePreview(mode config.Mode, preview Preview, confirmation Confirmation) Decision {
	requiresConfirmation := preview.Risk == string(sqlguard.HighRisk) || (mode == config.StrictMode && len(preview.SQL) > 0)

	if confirmation.Confirm {
		if confirmation.PreviewHash == "" || subtle.ConstantTimeCompare([]byte(confirmation.PreviewHash), []byte(preview.PreviewHash)) != 1 {
			return Decision{State: PreviewMismatch}
		}
		return Decision{State: ExecuteNow}
	}
	if requiresConfirmation {
		return Decision{State: PreviewRequired}
	}
	return Decision{State: ExecuteNow}
}
