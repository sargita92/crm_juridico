package application

import "strings"

// ResetCommand is the exact token a lead must send to reset the conversation.
const ResetCommand = "/reset"

// IsResetCommand returns true if any message in the batch, after TrimSpace,
// equals exactly ResetCommand. Case-sensitive by design so "resetando" doesn't
// trigger a reset.
func IsResetCommand(messages []string) bool {
	for _, m := range messages {
		if strings.TrimSpace(m) == ResetCommand {
			return true
		}
	}
	return false
}
