package lru

import (
	"reflect"
	"testing"
)

// These tests are from the old lmap repo

func TestCacheEviction(t *testing.T) {
	c := New[string, any](3)

	c.Add("one", 1)
	c.Add("two", 2.0)
	c.Add("three", byte(3))

	if v, ok := c.Get("one"); !ok || v != int(1) {
		t.Errorf("one: ok=%v, v=%v", ok, v)
	}
	if v, ok := c.Get("two"); !ok || v != float64(2.0) {
		t.Errorf("two: ok=%v, v=%v", ok, v)
	}
	if v, ok := c.Get("three"); !ok || v != byte(3) {
		t.Errorf("three: ok=%v, v=%v", ok, v)
	}
	t.Logf("%+v", c.l)

	c.Add("four", "4")
	if v, ok := c.Get("one"); ok || v != nil {
		t.Errorf("one (still present): ok=%v, v=%v", ok, v)
	}
	if v, ok := c.Get("two"); !ok || v != float64(2.0) {
		t.Errorf("two: ok=%v, v=%v", ok, v)
	}
	if v, ok := c.Get("three"); !ok || v != byte(3) {
		t.Errorf("three: ok=%v, v=%v", ok, v)
	}
	if v, ok := c.Get("four"); !ok || v != "4" {
		t.Errorf("four: ok=%v, v=%v", ok, v)
	}
	t.Logf("%+v", c.l)
}

func TestGetPeekUpdate(t *testing.T) {
	c := New[string, any](3)

	c.Add("one", 1)
	c.Add("two", 2.0)
	c.Add("three", byte(3))

	c.Get("one")

	c.Add("four", "4")
	if expected, keys := []string{"three", "one", "four"}, c.Keys(); !reflect.DeepEqual(expected, keys) {
		t.Errorf("keys: expected %v, got %v", expected, keys)
	}

	c.Peek("three")
	c.Add("five", 5+1i)
	if expected, keys := []string{"one", "four", "five"}, c.Keys(); !reflect.DeepEqual(expected, keys) {
		t.Errorf("keys: expected %v, got %v", expected, keys)
	}
}

func TestTrim(t *testing.T) {
	c := New[string, any](5)

	c.Add("one", 1)
	c.Add("two", 2.0)
	c.Add("three", byte(3))
	c.Add("four", "4")
	c.Add("five", 5+1i)

	if expected, keys := []string{"one", "two", "three", "four", "five"}, c.Keys(); !reflect.DeepEqual(expected, keys) {
		t.Errorf("keys: expected %v, got %v", expected, keys)
	}

	c.Trim(3)
	if expected, keys := []string{"three", "four", "five"}, c.Keys(); !reflect.DeepEqual(expected, keys) {
		t.Errorf("keys: expected %v, got %v", expected, keys)
	}
}
