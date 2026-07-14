package settings

import (
	"strconv"
	"time"
)

func ParseString(s string) (string, error) { return s, nil }

func ParseInt(s string) (int, error) { return strconv.Atoi(s) }

func ParseBool(s string) (bool, error) { return strconv.ParseBool(s) }

func ParseDuration(s string) (time.Duration, error) { return time.ParseDuration(s) }
