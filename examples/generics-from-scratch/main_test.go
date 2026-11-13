package main

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestGenericDemoOutput(t *testing.T) {
	// 捕获stdout的测试不t.Parallel；本节所有调用均同步执行。
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	os.Stdout = writer
	defer func() { os.Stdout = original }()
	main()
	os.Stdout = original
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	want := "泛型：最低值 3，结果 [3 5 7]\n泛型：最低值 6，结果 [7]\n泛型：前缀 \"Go\"，结果 [Go Gopher]\n"
	if string(data) != want {
		t.Fatalf("output=%q want=%q", data, want)
	}
}

func TestFilterPreservesOrderAndInput(t *testing.T) {
	items := []int{7, 1, 7, 3, 5}
	before := append([]int(nil), items...)
	seen := []int{}
	actual := filter(items, func(value int) bool {
		seen = append(seen, value)
		return value >= 3
	})
	if !reflect.DeepEqual(actual, []int{7, 7, 3, 5}) || !reflect.DeepEqual(seen, before) || !reflect.DeepEqual(items, before) {
		t.Fatalf("result=%v seen=%v input=%v", actual, seen, items)
	}
	actual[0] = 99
	if !reflect.DeepEqual(items, before) {
		t.Fatalf("result must not reuse input element storage: %v", items)
	}
}

func TestFilterEmptyAndAllOrNone(t *testing.T) {
	for _, input := range [][]int{nil, {}} {
		calls := 0
		actual := filter(input, func(int) bool { calls++; return true })
		if actual != nil || calls != 0 {
			t.Fatalf("empty result=%v callbacks=%d", actual, calls)
		}
	}
	input := []int{1, 2}
	all := filter(input, func(int) bool { return true })
	none := filter(input, func(int) bool { return false })
	if !reflect.DeepEqual(all, input) || none != nil {
		t.Fatalf("all=%v none=%v", all, none)
	}
	// 空输入不调用回调；非空输入仍要求调用方提供可调用的规则。
	var keep func(int) bool
	if filter([]int(nil), keep) != nil {
		t.Fatal("empty input must not call the predicate")
	}
}

func TestFilterMultipleElementTypes(t *testing.T) {
	stringsResult := filter([]string{"Go", "Java", "Gopher", "go"}, func(value string) bool {
		return strings.HasPrefix(value, "Go")
	})
	if !reflect.DeepEqual(stringsResult, []string{"Go", "Gopher"}) {
		t.Fatal(stringsResult)
	}
	type entry struct {
		Name       string
		Attributes map[string]int
	}
	// entry含map、不可比较；filter只搬运元素，不要求comparable。
	entries := []entry{{"a", map[string]int{"enabled": 1}}, {"b", map[string]int{"enabled": 0}}}
	actual := filter(entries, func(value entry) bool { return value.Attributes["enabled"] == 1 })
	if len(actual) != 1 || actual[0].Name != "a" {
		t.Fatal(actual)
	}
	// 结果元素是浅拷贝，不是深拷贝map或指针指向的数据。
	actual[0].Attributes["enabled"] = 2
	if entries[0].Attributes["enabled"] != 2 {
		t.Fatal("map references should retain Go's normal shallow-copy semantics")
	}
}

func TestFilterClosureUsesCurrentVariable(t *testing.T) {
	items := []int{1, 3, 5, 7}
	minimum := 3
	calls := 0
	keep := func(value int) bool { calls++; return value >= minimum }
	if calls != 0 {
		t.Fatal("creating function value must not invoke it")
	}
	first := filter(items, keep)
	minimum = 6
	second := filter[int](items, keep)
	if calls != 8 || !reflect.DeepEqual(first, []int{3, 5, 7}) || !reflect.DeepEqual(second, []int{7}) {
		t.Fatalf("calls=%d first=%v second=%v", calls, first, second)
	}
}

func TestFilterExplicitAndInferredTypes(t *testing.T) {
	items := []int{1, 2, 3}
	keep := func(value int) bool { return value > 1 }
	if !reflect.DeepEqual(filter(items, keep), filter[int](items, keep)) {
		t.Fatal("explicit and inferred instantiation must agree")
	}
}
