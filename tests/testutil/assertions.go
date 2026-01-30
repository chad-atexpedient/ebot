// Package testutil provides shared testing utilities
package testutil

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected: %v, Actual: %v", msg, expected, actual)
	}
}

// AssertNotEqual checks if two values are not equal
func AssertNotEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if reflect.DeepEqual(expected, actual) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected values to be different, but both are: %v", msg, expected)
	}
}

// AssertNil checks if a value is nil
func AssertNil(t *testing.T, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !isNil(actual) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected nil, but got: %v", msg, actual)
	}
}

// AssertNotNil checks if a value is not nil
func AssertNotNil(t *testing.T, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if isNil(actual) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected non-nil value", msg)
	}
}

// AssertTrue checks if a value is true
func AssertTrue(t *testing.T, actual bool, msgAndArgs ...interface{}) {
	t.Helper()
	if !actual {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected true, but got false", msg)
	}
}

// AssertFalse checks if a value is false
func AssertFalse(t *testing.T, actual bool, msgAndArgs ...interface{}) {
	t.Helper()
	if actual {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected false, but got true", msg)
	}
}

// AssertNoError checks if error is nil
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected no error, but got: %v", msg, err)
	}
}

// AssertError checks if error is not nil
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected error, but got nil", msg)
	}
}

// AssertErrorContains checks if error message contains a substring
func AssertErrorContains(t *testing.T, err error, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected error containing '%s', but got nil", msg, substr)
		return
	}
	if !strings.Contains(err.Error(), substr) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected error containing '%s', but got: %v", msg, substr, err)
	}
}

// AssertContains checks if a string contains a substring
func AssertContains(t *testing.T, s, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if !strings.Contains(s, substr) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected '%s' to contain '%s'", msg, s, substr)
	}
}

// AssertNotContains checks if a string does not contain a substring
func AssertNotContains(t *testing.T, s, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if strings.Contains(s, substr) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected '%s' to not contain '%s'", msg, s, substr)
	}
}

// AssertLen checks if a slice/map/string has the expected length
func AssertLen(t *testing.T, obj interface{}, expectedLen int, msgAndArgs ...interface{}) {
	t.Helper()
	actualLen := reflect.ValueOf(obj).Len()
	if actualLen != expectedLen {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected length %d, but got %d", msg, expectedLen, actualLen)
	}
}

// AssertEmpty checks if a slice/map/string is empty
func AssertEmpty(t *testing.T, obj interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	val := reflect.ValueOf(obj)
	if val.Len() != 0 {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected empty, but got %d elements", msg, val.Len())
	}
}

// AssertNotEmpty checks if a slice/map/string is not empty
func AssertNotEmpty(t *testing.T, obj interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	val := reflect.ValueOf(obj)
	if val.Len() == 0 {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected non-empty", msg)
	}
}

// AssertGreater checks if actual > expected
func AssertGreater(t *testing.T, actual, expected interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !isGreater(actual, expected) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected %v > %v", msg, actual, expected)
	}
}

// AssertGreaterOrEqual checks if actual >= expected
func AssertGreaterOrEqual(t *testing.T, actual, expected interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !isGreaterOrEqual(actual, expected) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected %v >= %v", msg, actual, expected)
	}
}

// AssertLess checks if actual < expected
func AssertLess(t *testing.T, actual, expected interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !isLess(actual, expected) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected %v < %v", msg, actual, expected)
	}
}

// AssertLessOrEqual checks if actual <= expected
func AssertLessOrEqual(t *testing.T, actual, expected interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !isLessOrEqual(actual, expected) {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected %v <= %v", msg, actual, expected)
	}
}

// AssertInDelta checks if two floats are within delta of each other
func AssertInDelta(t *testing.T, expected, actual, delta float64, msgAndArgs ...interface{}) {
	t.Helper()
	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}
	if diff > delta {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected %f to be within %f of %f", msg, actual, delta, expected)
	}
}

// AssertWithinDuration checks if two times are within duration of each other
func AssertWithinDuration(t *testing.T, expected, actual time.Time, delta time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	diff := expected.Sub(actual)
	if diff < 0 {
		diff = -diff
	}
	if diff > delta {
		msg := formatMessage(msgAndArgs...)
		t.Errorf("%sExpected %v to be within %v of %v", msg, actual, delta, expected)
	}
}

// AssertPanics checks if a function panics
func AssertPanics(t *testing.T, f func(), msgAndArgs ...interface{}) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			msg := formatMessage(msgAndArgs...)
			t.Errorf("%sExpected function to panic", msg)
		}
	}()
	f()
}

// AssertNotPanics checks if a function does not panic
func AssertNotPanics(t *testing.T, f func(), msgAndArgs ...interface{}) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			msg := formatMessage(msgAndArgs...)
			t.Errorf("%sExpected function not to panic, but panicked with: %v", msg, r)
		}
	}()
	f()
}

// AssertEventually checks if a condition becomes true within timeout
func AssertEventually(t *testing.T, condition func() bool, timeout, interval time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}
	msg := formatMessage(msgAndArgs...)
	t.Errorf("%sCondition not satisfied within %v", msg, timeout)
}

// AssertNever checks if a condition never becomes true within duration
func AssertNever(t *testing.T, condition func() bool, duration, interval time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		if condition() {
			msg := formatMessage(msgAndArgs...)
			t.Errorf("%sCondition was satisfied but should not have been", msg)
			return
		}
		time.Sleep(interval)
	}
}

// Helper functions

func formatMessage(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 {
		return ""
	}
	if len(msgAndArgs) == 1 {
		return msgAndArgs[0].(string) + ": "
	}
	return strings.TrimRight(msgAndArgs[0].(string), ": ") + ": "
}

func isNil(obj interface{}) bool {
	if obj == nil {
		return true
	}
	val := reflect.ValueOf(obj)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return val.IsNil()
	}
	return false
}

func isGreater(a, b interface{}) bool {
	switch av := a.(type) {
	case int:
		return av > b.(int)
	case int64:
		return av > b.(int64)
	case float64:
		return av > b.(float64)
	case string:
		return av > b.(string)
	case time.Duration:
		return av > b.(time.Duration)
	}
	return false
}

func isGreaterOrEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b) || isGreater(a, b)
}

func isLess(a, b interface{}) bool {
	switch av := a.(type) {
	case int:
		return av < b.(int)
	case int64:
		return av < b.(int64)
	case float64:
		return av < b.(float64)
	case string:
		return av < b.(string)
	case time.Duration:
		return av < b.(time.Duration)
	}
	return false
}

func isLessOrEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b) || isLess(a, b)
}
