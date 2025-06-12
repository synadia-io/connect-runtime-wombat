package compiler

import (
	"testing"

	"github.com/synadia-io/connect-runtime-wombat/utils"
)

func TestFragment(t *testing.T) {
	tests := []struct {
		name string
		frag Fragment
		exp  map[string]any
	}{
		{"should set strings", Frag().String("name", "calmera"), map[string]any{"name": "calmera"}},
		{"should set string pointers", Frag().StringP("name", utils.Ptr("calmera")), map[string]any{"name": "calmera"}},
		{"should set ints", Frag().Int("age", 42), map[string]any{"age": 42}},
		{"should set int pointers", Frag().IntP("age", utils.Ptr(42)), map[string]any{"age": 42}},
		{"should set bools", Frag().Bool("active", true), map[string]any{"active": true}},
		{"should set bool pointers", Frag().BoolP("active", utils.Ptr(true)), map[string]any{"active": true}},
		{"should set sub fragments", Frag().Fragment("sub", Frag().String("name", "calmera")), map[string]any{"sub": map[string]any{"name": "calmera"}}},
		{"should set multiple values", Frag().String("name", "calmera").Int("age", 42), map[string]any{"name": "calmera", "age": 42}},
		{"should overwrite values", Frag().String("name", "calmera").String("name", "daan"), map[string]any{"name": "daan"}},
		{"should overwrite values with pointers", Frag().String("name", "calmera").StringP("name", utils.Ptr("daan")), map[string]any{"name": "daan"}},
		{"should set multiple sub fragments", Frag().Fragments("sub", Frag().String("name", "calmera"), Frag().Int("age", 42)), map[string]any{"sub": []Fragment{map[string]any{"name": "calmera"}, map[string]any{"age": 42}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.frag.EqualsMap(tt.exp) {
				t.Errorf("expected %v, got %v", tt.exp, tt.frag)
			}
		})
	}
}
