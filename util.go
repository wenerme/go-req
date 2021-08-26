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
		c[k] = append(v, c[k]...)
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
