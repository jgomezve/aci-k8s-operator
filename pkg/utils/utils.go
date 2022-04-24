package utils

// TODO: Use new Go generics
// Check if element exists in slice
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Remove item from slice
func Remove(l []string, item string) []string {
	for i, other := range l {
		if other == item {
			return append(l[:i], l[i+1:]...)
		}
	}
	return l
}

// Intersection of two slice
func Intersect(a, b []string) []string {
	inter := []string{}
	hash := make(map[string]bool)
	for _, e := range a {
		hash[e] = true
	}
	for _, e := range b {
		if hash[e] {
			inter = append(inter, e)
		}
	}
	return inter
}

// Unique values in one slice
func Unique(a, uq []string) []string {
	unique := []string{}
	hash := make(map[string]bool)
	for _, e := range a {
		hash[e] = true
	}
	for _, e := range uq {
		if !hash[e] {
			unique = append(unique, e)
		}
	}
	return unique
}

func ToStringList(configured []interface{}) []string {
	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		val, ok := v.(string)
		if ok && val != "" {
			vs = append(vs, val)
		}
	}
	return vs
}
