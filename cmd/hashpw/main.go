package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sakid00/massmaker-be/internal/auth"
)

func main() {
	check := flag.String("check", "", "bcrypt hash to verify against the password on stdin")
	flag.Parse()

	plain, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read password: %v\n", err)
		os.Exit(1)
	}
	password := strings.TrimRight(string(plain), "\r\n")
	if password == "" {
		fmt.Fprintln(os.Stderr, "password is required on stdin")
		os.Exit(1)
	}

	if *check != "" {
		if !auth.Verify(*check, password) {
			os.Exit(1)
		}
		return
	}

	hash, err := auth.Hash(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash password: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(hash)
}
