package utils

import (
	"bufio"
	"fmt"
	"os"
)

func AskValue(question string) string {
	reader := bufio.NewReader(os.Stdin)
	scanner := bufio.NewScanner(reader)
	fmt.Print(question)
	scanner.Scan()
	return scanner.Text()
}
