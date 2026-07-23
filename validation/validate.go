package validation

// Validate runs all rules and returns message keys for every failure.
// It does not stop at the first failure.
func Validate(value string, rules ...Rule) []string {
	var keys []string
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		ok, key := rule(value)
		if !ok {
			keys = append(keys, key)
		}
	}
	return keys
}
