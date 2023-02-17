package util

import "testing"

func TestSet(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		var v int
		var i interface{} = 10
		Set(i, &v)
		if v != 10 {
			t.Errorf("want 10 got %d", v)
		}
	})
	t.Run("int pointer", func(t *testing.T) {
		var p *int
		v := 10
		var i interface{} = &v
		Set(i, &p)
		if p == nil {
			t.Errorf("t is nil")
		} else if *p != 10 {
			t.Errorf("want 10 got %d", *p)
		}
	})
	t.Run("string", func(t *testing.T) {
		var s string
		var i interface{} = "hello"
		Set(i, &s)
		if s != "hello" {
			t.Errorf("want hello got %s", s)
		}
	})
	t.Run("string to int", func(t *testing.T) {
		var v int
		var i interface{} = "hello"
		Set(i, &v)
		if v != 0 {
			t.Errorf("want 0 got %d", v)
		}
	})
}

func TestHidePhone(t *testing.T) {
	t.Run("正常", func(t *testing.T) {
		phone := "13344445555"
		want := "*******5555"
		ret := HidePhone(phone)
		if ret != want {
			t.Errorf("want %s got %s", want, ret)
		}
	})
	t.Run("带国家符号", func(t *testing.T) {
		phone := "+8613344445555"
		want := "**********5555"
		ret := HidePhone(phone)
		if ret != want {
			t.Errorf("want %s got %s", want, ret)
		}
	})
	t.Run("固话", func(t *testing.T) {
		phone := "8899666"
		want := "***9666"
		ret := HidePhone(phone)
		if ret != want {
			t.Errorf("want %s got %s", want, ret)
		}
	})
	t.Run("三巨头", func(t *testing.T) {
		phone := "10086"
		want := "*0086"
		ret := HidePhone(phone)
		if ret != want {
			t.Errorf("want %s got %s", want, ret)
		}
	})
	t.Run("短小", func(t *testing.T) {
		phone := "1000"
		want := "1000"
		ret := HidePhone(phone)
		if ret != want {
			t.Errorf("want %s got %s", want, ret)
		}
	})
}
