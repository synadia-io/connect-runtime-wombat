package compiler

import (
	"bytes"
	"encoding/json"
)

func Frag() Fragment {
	return make(Fragment)
}

type Fragment map[string]any

func (f Fragment) Fragment(key string, fragment Fragment) Fragment {
	f[key] = fragment
	return f
}

func (f Fragment) StringMap(key string, m map[string]string) Fragment {
	f[key] = m
	return f
}

func (f Fragment) Map(key string, m map[string]any) Fragment {
	f[key] = m
	return f
}

func (f Fragment) Fragments(key string, fragments ...Fragment) Fragment {
	f[key] = fragments
	return f
}

func (f Fragment) Strings(key string, values ...string) Fragment {
	f[key] = values
	return f
}

func (f Fragment) String(key string, value string) Fragment {
	f[key] = value
	return f
}

func (f Fragment) StringP(key string, value *string) Fragment {
	if value != nil {
		f[key] = *value
	}
	return f
}

func (f Fragment) Int(key string, value int) Fragment {
	f[key] = value
	return f
}

func (f Fragment) IntP(key string, value *int) Fragment {
	if value != nil {
		f[key] = *value
	}
	return f
}

func (f Fragment) Bool(key string, value bool) Fragment {
	f[key] = value
	return f
}

func (f Fragment) BoolP(key string, value *bool) Fragment {
	if value != nil {
		f[key] = *value
	}
	return f
}

func (f Fragment) EqualsMap(exp map[string]any) bool {
	b1, _ := json.Marshal(f)
	b2, _ := json.Marshal(exp)
	return bytes.Equal(b1, b2)
}
