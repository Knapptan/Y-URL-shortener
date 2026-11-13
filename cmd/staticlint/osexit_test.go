package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestNoOSExit проверяет работу пользовательского анализатора noosexit,
// который запрещает вызов os.Exit в функции main пакета main.
func TestNoOSExit(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noosexit, "mynoosexit/good")
}

// TestNoOSExitErrors проверяет, что анализатор корректно
// выявляет нарушения правила.
func TestNoOSExitErrors(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noosexit, "mynoosexit/bad")
}
