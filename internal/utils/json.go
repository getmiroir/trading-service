package utils

import (
	"encoding/json"
	"strconv"
)

// Float64String is a float64 that can be unmarshaled from both string and number
type Float64String float64

// UnmarshalJSON implements json.Unmarshaler to handle both string and number
func (f *Float64String) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "" {
			*f = 0
			return nil
		}
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		*f = Float64String(val)
		return nil
	}
	// If not a string, try as number
	var n float64
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*f = Float64String(n)
	return nil
}

// Float64 returns the float64 value
func (f Float64String) Float64() float64 {
	return float64(f)
}

// Int64String is an int64 that can be unmarshaled from both string and number
type Int64String int64

// UnmarshalJSON implements json.Unmarshaler to handle both string and number
func (i *Int64String) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "" {
			*i = 0
			return nil
		}
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*i = Int64String(val)
		return nil
	}
	// If not a string, try as number
	var n int64
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*i = Int64String(n)
	return nil
}

// Int64 returns the int64 value
func (i Int64String) Int64() int64 {
	return int64(i)
}

// BoolString is a bool that can be unmarshaled from both string and boolean
type BoolString bool

// UnmarshalJSON implements json.Unmarshaler to handle both string and boolean
func (b *BoolString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*b = BoolString(s == "true")
		return nil
	}
	var n bool
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*b = BoolString(n)
	return nil
}

// Bool returns the bool value
func (b BoolString) Bool() bool {
	return bool(b)
}
