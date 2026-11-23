package utils

// Changes returns the items of after that before lacks and the items of before that after lacks, matching items by key
func Changes[T any](before, after []T, key func(T) string) (added, removed []T) {
	beforeKeys := make(map[string]struct{}, len(before))
	for _, item := range before {
		beforeKeys[key(item)] = struct{}{}
	}
	afterKeys := make(map[string]struct{}, len(after))
	for _, item := range after {
		afterKeys[key(item)] = struct{}{}
		if _, ok := beforeKeys[key(item)]; !ok {
			added = append(added, item)
		}
	}
	for _, item := range before {
		if _, ok := afterKeys[key(item)]; !ok {
			removed = append(removed, item)
		}
	}
	return added, removed
}
