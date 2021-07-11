package req

func mergeMapSliceString(a map[string][]string, b map[string][]string) map[string][]string {
	if len(a) == 0 {
		return cloneMapSliceString(b)
	}
	if len(b) == 0 {
		return cloneMapSliceString(a)
	}
	c := cloneMapSliceString(a)
	for k, v := range b {
		c[k] = append(c[k], v...)
	}
	return c
}

func cloneMapSliceString(h map[string][]string) map[string][]string {
	// return http.Header(h).Clone()
	if h == nil {
		return nil
	}

	// Find total number of values.
	nv := 0
	for _, vv := range h {
		nv += len(vv)
	}
	sv := make([]string, nv) // shared backing array for headers' values
	h2 := make(map[string][]string, len(h))
	for k, vv := range h {
		n := copy(sv, vv)
		h2[k] = sv[:n:n]
		sv = sv[n:]
	}
	return h2
}

// Values represent a map slice
type Values map[string][]string

// Get gets the first value associated with the given key.
// If there are no values associated with the key, Get returns
// the empty string. To access multiple values, use the map
// directly.
func (v Values) Get(key string) string {
	if v == nil {
		return ""
	}
	vs := v[key]
	if len(vs) == 0 {
		return ""
	}
	return vs[0]
}

// Set sets the key to value. It replaces any existing
// values.
func (v Values) Set(key, value string) Values {
	v[key] = []string{value}
	return v
}

// Add adds the value to key. It appends to any existing
// values associated with key.
func (v Values) Add(key, value string) Values {
	v[key] = append(v[key], value)
	return v
}

// Del deletes the values associated with key.
func (v Values) Del(key string) Values {
	delete(v, key)
	return v
}

func (v Values) Clone() Values {
	return cloneMapSliceString(v)
}

func (v Values) WithMerge(o Values) Values {
	for k, vv := range o {
		v[k] = append(v[k], vv...)
	}
	return v
}

func (v Values) WithOverride(o Values) Values {
	for k, vv := range o {
		v[k] = vv
	}
	return v
}
