package provider

import "fmt"

func buildUserMessage(req CommandRequest) string {
	return fmt.Sprintf("OS: %s\nShell: %s\nQuery: %s", req.OS, req.Shell, req.Query)
}
