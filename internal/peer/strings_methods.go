package peer

import (
	mathrand "math/rand"
	"strings"
	"time"
)

func (peer *Peer) removeDuplicates(slice []string) []string {
	allKeys := make(map[string]bool)

	resultSlice := make([]string, 0)

	for _, value := range slice {
		if _, v := allKeys[value]; !v {
			allKeys[value] = true
			resultSlice = append(resultSlice, value)
		}
	}

	return resultSlice
}

func (peer *Peer) removeElement(slice []string, idx int) []string {
	slice[idx] = slice[len(slice)-1]
	return slice[:len(slice)-1]
}

func (peer *Peer) lowerUpper(str string) string {
	var res string

	mathrand.Seed(time.Now().UnixNano())

	for i := 0; i < len(str); i++ {
		a := mathrand.Intn(100)

		if a < 50 {
			newStr := strings.ToUpper(string(str[i]))
			res += newStr
		} else {
			newStr := strings.ToLower(string(str[i]))
			res += newStr
		}
	}
	return res
}
