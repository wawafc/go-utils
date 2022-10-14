package clog

import (
	"encoding/json"
	"github.com/rs/zerolog"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func decodeIfBinaryToBytes(in []byte) []byte {
	return in
}

func toTime(i interface{}) time.Time {
	var t time.Time
	switch tt := i.(type) {
	case string:
		ts, err := time.Parse(zerolog.TimeFieldFormat, tt)
		if err != nil {
			panic("toTime cannot parse time")
		} else {
			t = ts
		}
	case json.Number:
		i, err := tt.Int64()
		if err != nil {
			panic("toTime cannot parse json.Number to int64")
		} else {
			var sec, nsec int64 = i, 0
			switch zerolog.TimeFieldFormat {
			case zerolog.TimeFormatUnixMs:
				nsec = int64(time.Duration(i) * time.Millisecond)
				sec = 0
			case zerolog.TimeFormatUnixMicro:
				nsec = int64(time.Duration(i) * time.Microsecond)
				sec = 0
			}
			ts := time.Unix(sec, nsec)
			t = ts
		}
	}
	return t
}

func toMBSize(maxSize string) int64 {
	maxSize = strings.TrimSpace(maxSize)
	if maxSize == "" {
		return 0
	}

	var isize, iunit []int
	var size, unit string
	for i, c := range maxSize {
		switch {
		case unicode.IsNumber(c):
			isize = append(isize, i)
			continue
		case unicode.IsLetter(c):
			iunit = append(iunit, i)
		}
	}

	for i, v := range isize {
		if i != 0 && isize[i] != isize[i-1]+1 {
			panic("invalid MaxSize")
		}
		size += string(maxSize[v])
	}
	for i, v := range iunit {
		if i != 0 && iunit[i] != iunit[i-1]+1 {
			panic("invalid MaxSize")
		}
		unit += string(maxSize[v])
	}

	if size == "" {
		panic("invalid MaxSize: size is not found")
	}

	intSize, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		panic("cannot convert size to int")
	}

	var mbSize int64
	switch unit {
	case "":
		mbSize = bytesUnit * intSize
	case "k", "K", "KB", "kb":
		mbSize = kilobytes * intSize
	case "m", "M", "MB", "mb":
		mbSize = megabytes * intSize
	case "g", "G", "GB", "gb":
		mbSize = gigabytes * intSize
	default:
		panic("invalid MaxSize: size is invalid")
	}

	return mbSize
}
