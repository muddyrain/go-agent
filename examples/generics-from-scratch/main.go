package main

import (
	"fmt"
	"strings"
)

func filter[T any](items []T, keep func(T) bool) []T {
	var result []T
	for _, item := range items {
		// 筛选规则由调用方提供；按输入顺序保留，不改写输入切片。
		if keep(item) {
			result = append(result, item)
		}
	}
	return result
}

func main() {
	numbers := []int{1, 3, 5, 7}
	minimum := 3
	// 创建函数值时不执行；每次调用时读取同一个minimum变量。
	keepNumber := func(value int) bool {
		return value >= minimum
	}
	selected := filter(numbers, keepNumber)
	fmt.Printf("泛型：最低值 %d，结果 %v\n", minimum, selected)

	minimum = 6
	selected = filter[int](numbers, keepNumber)
	fmt.Printf("泛型：最低值 %d，结果 %v\n", minimum, selected)

	words := []string{"Go", "Java", "Gopher", "go"}
	prefix := "Go"
	selectedWords := filter(words, func(word string) bool {
		return strings.HasPrefix(word, prefix)
	})
	fmt.Printf("泛型：前缀 %q，结果 %v\n", prefix, selectedWords)
}
