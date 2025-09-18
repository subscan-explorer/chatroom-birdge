package utils

import (
	"strconv"
	"strings"
	"time"
)

func Unique[T comparable](param []T) []T {
	set := make(map[T]struct{})
	var result []T
	for _, v := range param {
		if _, exist := set[v]; exist {
			continue
		}
		result = append(result, v)
		set[v] = struct{}{}
	}
	return result
}

func Default[T any](v T, judge func(v T) bool, df T) T {
	if judge(v) {
		return v
	}
	return df
}

func IfElse[T any](b bool, x T, y T) T {
	if b {
		return x
	}
	return y
}

func ParseSlackTimestamp(ts string) int64 {
	if len(ts) == 0 {
		return 0
	}
	tm := strings.Split(ts, ".")
	switch len(tm) {
	case 2:
		sec, err := strconv.ParseInt(tm[0], 10, 64)
		if err != nil {
			return 0
		}
		nsec, err := strconv.ParseInt(tm[1]+strings.Repeat("0", 9-len(tm[1])), 10, 64)
		if err != nil {
			return 0
		}
		return time.Unix(sec, nsec).UnixNano()
	case 1:
		sec, err := strconv.ParseInt(tm[0], 10, 64)
		if err != nil {
			return 0
		}
		return time.Unix(sec, 0).UnixNano()
	default:
		return 0
	}
}

func FilterSlice[T any](s []T, isFilter func(T) bool) []T {
	var ns []T
	for _, v := range s {
		if !isFilter(v) {
			ns = append(ns, v)
		}
	}
	return ns
}

func Map[T any, S any](data []S, f func(v S) T) []T {
	var result = make([]T, 0, len(data))
	for _, datum := range data {
		result = append(result, f(datum))
	}
	return result
}

func MapKeyToSlice[K comparable, V any](m map[K]V) []K {
	result := make([]K, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
