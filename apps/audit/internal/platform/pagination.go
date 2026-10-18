package platform

import "strconv"

// ParsePagination разбирает limit/offset и проверяет границы.
// Общее правило для всех списков: limit 1..100, offset 0..100000.
func ParsePagination(limitParam, offsetParam string) (limit, offset int, ok bool) {
	limit, err := strconv.Atoi(limitParam)
	offset, offsetErr := strconv.Atoi(offsetParam)
	if err != nil || offsetErr != nil || limit < 1 || limit > 100 || offset < 0 || offset > 100000 {
		return 0, 0, false
	}
	return limit, offset, true
}
