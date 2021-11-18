package mt

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
)

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

func expectStatus(expected, actual int) error {
	if expected != actual {
		return fmt.Errorf(`expected status %d, got %d`, expected, actual)
	}
	return nil
}

func expect(key string, expected, actual interface{}, exactJSON bool) []error {
	switch expectedValue := expected.(type) {

	case Object, map[string]interface{}:
		ev, ok := expectedValue.(map[string]interface{})
		if !ok {
			ev = map[string]interface{}(expectedValue.(Object))
		}
		return expectObject(key, ev, actual, exactJSON)

	case Array, []interface{}:
		ev, ok := expectedValue.([]interface{})
		if !ok {
			ev = []interface{}(expectedValue.(Array))
		}
		return expectArray(key, ev, actual, exactJSON)

	case string:
		err := expectString(key, expectedValue, actual)
		if err != nil {
			return []error{err}
		}

	case float64:
		err := expectNumber(key, expectedValue, actual)
		if err != nil {
			return []error{err}
		}

	case int, int64:
		ev, ok := expectedValue.(int64)
		if !ok {
			ev = int64(expectedValue.(int))
		}

		err := expectNumber(key, float64(ev), actual)
		if err != nil {
			return []error{err}
		}

	case bool:
		err := expectBool(key, expectedValue, actual)
		if err != nil {
			return []error{err}
		}

	case func(interface{}) bool:
		if !expectedValue(actual) {
			return []error{fmt.Errorf("field %q did not satisfy predicate, got %q", key, actual)}
		}

	default:
		return []error{fmt.Errorf("unexpected value type for field %q: %T", key, actual)}
	}

	return nil
}

func expectBool(key string, expected bool, actual interface{}) error {
	b, ok := actual.(bool)
	if !ok {
		return wrongTypeError(key, expected, actual)
	}

	if b != expected {
		return wrongValueError(key, expected, actual)
	}

	return nil
}

func expectNumber(key string, expected float64, actual interface{}) error {
	n, ok := actual.(float64)
	if !ok {
		return wrongTypeError(key, expected, actual)
	}

	if n != expected {
		return wrongValueError(key, expected, actual)
	}

	return nil
}

func expectString(key string, expected string, actual interface{}) error {
	s, ok := actual.(string)
	if !ok {
		return wrongTypeError(key, expected, actual)
	}

	if s != expected {
		return wrongValueError(key, expected, actual)
	}

	return nil
}

func expectObject(key string, expected map[string]interface{}, actual interface{}, exact bool) []error {
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
		if elemErrs := expect(fmt.Sprintf("%s.%s", key, k), v, m[k], exact); len(elemErrs) > 0 {
			errs = append(errs, elemErrs...)
		}
	}

	return errs
}

func expectArray(key string, expected []interface{}, actual interface{}, exact bool) []error {
	a, ok := actual.([]interface{})
	if !ok {
		return []error{wrongTypeError(key, expected, actual)}
	}

	if exact && len(a) != len(expected) {
		return []error{fmt.Errorf("expected %d elements, got %d", len(expected), len(a))}
	}

	errs := []error{}
	for i, v := range expected {
		if elemErrs := expect(fmt.Sprintf("%s[%d]", key, i), v, a[i], exact); len(elemErrs) > 0 {
			errs = append(errs, elemErrs...)
		}
	}

	return errs
}

func expectHeaders(expected http.Header, actual http.Header) []error {
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