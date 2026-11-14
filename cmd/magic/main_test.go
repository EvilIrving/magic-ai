package main

import (
	"os"
	"testing"

	"magic/internal/testenv"
)

func TestMain(m *testing.M) { os.Exit(testenv.Main(m)) }
