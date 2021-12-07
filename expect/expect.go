package expect

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"

	mtjson "github.com/jefflinse/melatonin/json"
)

// A CustomPredicate is a function that takes a test result value and possibly returns an error.
type CustomPredicate func(interface{}) error

// A CustomPredicateForKey is a function that takes a key and a value and possibly returns an error.
type CustomPredicateForKey func(string, interface{}) error

// Bind binds a value to the provided target variable.
func Bind(target interface{}) CustomPredicateForKey {
	return func(key string, actual interface{}) error {

		switch typedTarget := target.(type) {

		case *string:
			typedActual, ok := actual.(string)
			if !ok {
				return fmt.Errorf("expected to bind string, found %T", actual)
			}

			*typedTarget = typedActual

		case *float64:
			typedActual, ok := actual.(float64)
			if !ok {
				return fmt.Errorf("expected to bind float64, found %T", actual)
			}

			*typedTarget = typedActual

		case *int64:
			typedActual, ok := actual.(int64)
			if !ok {
				return fmt.Errorf("expected to bind int64, found %T", actual)
			}

			*typedTarget = typedActual

		case *bool:
			typedActual, ok := actual.(bool)
			if !ok {
				return fmt.Errorf("expected to bind bool, found %T", actual)
			}

			*typedTarget = typedActual

		// struct, map, or slice
		default:
			b, err := json.Marshal(actual)
			if err != nil {
				return fmt.Errorf("failed to bind %T to %T: %s", actual, typedTarget, err)
			}

			if err := json.Unmarshal(b, target); err != nil {
				return fmt.Errorf("failed to bind %T to %T: %s", actual, typedTarget, err)
			}
		}

		return nil
	}
}

// Predicate runs a custom predicate against an value.
func Predicate(fn CustomPredicate) CustomPredicateForKey {
	return func(key string, actual interface{}) error {
		if err := fn(actual); err != nil {
			return failedPredicateError(key, err)
		}

		return nil
	}
}

// Headers compares a set of expected headers against a set of actual headers,
func Headers(expected http.Header, actual http.Header) []error {
	var errs []error
	for key, expectedValuesForKey := range expected {
		actualValuesForKey, ok := actual[key]
		if !ok {
			errs = append(errs, fmt.Errorf("expected header %q, got nothing", key))
			continue
		}

		sort.Strings(expectedValuesForKey)
		sort.Strings(actualValuesForKey)

		for _, expectedValue := range expectedValuesForKey {
			found := false
			for _, actualValue := range actualValuesForKey {
				if actualValue == expectedValue {
					found = true
					break
				}
			}

			if !found {
				errs = append(errs, fmt.Errorf("expected header %q to contain %q, got %q", key, expectedValue, actualValuesForKey))
			}
		}
	}

	return errs
}

// Status compares an expected status code to an actual status code.
func Status(expected, actual int) error {
	if expected != actual {
		return fmt.Errorf(`expected status %d, got %d`, expected, actual)
	}
	return nil
}

// ValueForKey compares an expected value to an actual value.
func ValueForKey(key string, expected, actual interface{}, exactJSON bool) []error {
	switch expectedValue := expected.(type) {

	case mtjson.Object, map[string]interface{}:
		ev, ok := expectedValue.(map[string]interface{})
		if !ok {
			ev = map[string]interface{}(expectedValue.(mtjson.Object))
		}
		return mapValForKey(key, ev, actual, exactJSON)

	case mtjson.Array, []interface{}:
		ev, ok := expectedValue.([]interface{})
		if !ok {
			ev = []interface{}(expectedValue.(mtjson.Array))
		}
		return arrayValForKey(key, ev, actual, exactJSON)

	case string:
		err := strValForKey(key, expectedValue, actual)
		if err != nil {
			return []error{err}
		}

	case *string:
		return []error{errors.New("bar")}

	case float64:
		err := numValForKey(key, expectedValue, actual)
		if err != nil {
			return []error{err}
		}

	case *float64:
		err := numValForKey(key, *expectedValue, actual)
		if err != nil {
			return []error{err}
		}

	case int, int64:
		ev, ok := expectedValue.(int64)
		if !ok {
			ev = int64(expectedValue.(int))
		}

		err := numValForKey(key, float64(ev), actual)
		if err != nil {
			return []error{err}
		}

	case *int, *int64:
		return []error{errors.New("foo")}

	case bool:
		err := boolValForKey(key, expectedValue, actual)
		if err != nil {
			return []error{err}
		}

	case CustomPredicateForKey:
		if err := expectedValue(key, actual); err != nil {
			return []error{err}
		}

	default:
		return []error{fmt.Errorf("unexpected value type for field %q: %T", key, actual)}
	}

	return nil
}

// boolValForKey compares an expected bool to an actual bool.
func boolValForKey(key string, expected bool, actual interface{}) error {
	b, ok := actual.(bool)
	if !ok {
		return wrongTypeError(key, expected, actual)
	}

	if b != expected {
		return wrongValueError(key, expected, actual)
	}

	return nil
}

// numValForKey compares an expected float64 to an actual float64.
func numValForKey(key string, expected float64, actual interface{}) error {
	n, ok := actual.(float64)
	if !ok {
		return wrongTypeError(key, expected, actual)
	}

	if n != expected {
		return wrongValueError(key, expected, actual)
	}

	return nil
}

// strValForKey compares an expected string to an actual string.
func strValForKey(key string, expected string, actual interface{}) error {
	s, ok := actual.(string)
	if !ok {
		return wrongTypeError(key, expected, actual)
	}

	if s != expected {
		return wrongValueError(key, expected, actual)
	}

	return nil
}

// mapValForKey compares an expected JSON object to an actual JSON object.
func mapValForKey(key string, expected map[string]interface{}, actual interface{}, exact bool) []error {
	m, ok := actual.(map[string]interface{})
	if !ok {
		return []error{wrongTypeError(key, expected, actual)}
	}

	if exact {
		if len(m) != len(expected) {
			return []error{fmt.Errorf("expected %d fields, got %d", len(expected), len(m))}
		}

		expectedKeys := make([]string, 0, len(expected))
		for k := range expected {
			expectedKeys = append(expectedKeys, k)
		}

		actualKeys := make([]string, 0, len(m))
		for k := range m {
			actualKeys = append(actualKeys, k)
		}

		sort.Strings(expectedKeys)
		sort.Strings(actualKeys)

		for i := range expectedKeys {
			if expectedKeys[i] != actualKeys[i] {
				return []error{fmt.Errorf("expected key %q, got %q", expectedKeys[i], actualKeys[i])}
			}
		}
	}

	errs := []error{}
	for k, v := range expected {
		if elemErrs := ValueForKey(fmt.Sprintf("%s.%s", key, k), v, m[k], exact); len(elemErrs) > 0 {
			errs = append(errs, elemErrs...)
		}
	}

	return errs
}

// arrayValForKey compares an expected JSON array to an actual JSON array.
func arrayValForKey(key string, expected []interface{}, actual interface{}, exact bool) []error {
	a, ok := actual.([]interface{})
	if !ok {
		return []error{wrongTypeError(key, expected, actual)}
	}

	if exact && len(a) != len(expected) {
		return []error{fmt.Errorf("expected %d elements, got %d", len(expected), len(a))}
	}

	errs := []error{}
	for i, v := range expected {
		if elemErrs := ValueForKey(fmt.Sprintf("%s[%d]", key, i), v, a[i], exact); len(elemErrs) > 0 {
			errs = append(errs, elemErrs...)
		}
	}

	return errs
}

func failedPredicateError(key string, err error) error {
	msg := fmt.Sprintf("predicate failed: %s", err)
	if key != "" {
		msg = fmt.Sprintf("%s: %s", key, msg)
	}

	return errors.New(msg)
}

func wrongTypeError(key string, expected, actual interface{}) error {
	var msg string
	if expected != nil && actual == nil {
		msg = fmt.Sprintf("expected %T, got nothing", expected)
	} else {
		msg = fmt.Sprintf(`expected type "%T", got '%T"`, expected, actual)
	}

	if key != "" {
		msg = fmt.Sprintf("%s: %s", key, msg)
	}

	return errors.New(msg)
}

func wrongValueError(key string, expected, actual interface{}) error {
	var msg string
	if expected != nil && actual == nil {
		msg = fmt.Sprintf("expected %v, got nothing", expected)
	} else {
		msg = fmt.Sprintf(`expected "%v", got "%v"`, expected, actual)
	}

	if key != "" {
		msg = fmt.Sprintf("%s: %s", key, msg)
	}

	return errors.New(msg)
}
