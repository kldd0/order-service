package utils

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func GenerateUID19() string {
	n := 19
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	s := fmt.Sprintf("%x", b)
	return s
}

func GenerateUID19v2() string {
	uuid := uuid.NewString()
	s := strings.ReplaceAll(uuid, "-", "")
	return s[:19]
}
