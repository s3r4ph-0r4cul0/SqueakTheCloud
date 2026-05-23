package output

import "fmt"

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
)

func LogInfo(msg string) {
	fmt.Printf("%s[*] %s%s\n", Cyan, msg, Reset)
}

func LogSuccess(msg string) {
	fmt.Printf("%s[+] %s%s\n", Green, msg, Reset)
}

func LogWarning(msg string) {
	fmt.Printf("%s[!] %s%s\n", Yellow, msg, Reset)
}

func LogCritical(msg string) {
	fmt.Printf("%s[CRITICAL] %s%s\n", Red, msg, Reset)
}
