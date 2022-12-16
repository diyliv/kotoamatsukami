package client

import (
	"bufio"
	"os"
	"strings"
)

func InputString() string {
	msg, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		panic(err)
	}

	return strings.Replace(msg, "\n", "", -1)
}
