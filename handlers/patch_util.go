package handlers

import "strings"

// camelToSnake converts a camelCase string (as produced by Dart's
// `toJson()`, e.g. "targetAmount") into the snake_case form GORM uses by
// default for DB column names (e.g. "target_amount").
func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// camelKeysToSnake rewrites every key of a JSON-decoded patch map from
// camelCase to snake_case.
//
// Why this is needed: PATCH handlers bind the request body into a raw
// map[string]interface{} so partial updates work without a rigid struct.
// GORM's Updates() accepts such a map, but — unlike passing a typed
// struct — it does NOT run the key through the schema/naming strategy to
// find the matching DB column; it uses the key as the literal SQL column
// name. Our Flutter client always sends camelCase keys straight from
// Dart's `toJson()` (e.g. "targetAmount", "targetDate"), while the actual
// MySQL columns are snake_case (e.g. "target_amount", "target_date").
// Left unconverted, MySQL rejects the whole UPDATE statement with
// "Unknown column '<key>' in 'SET'" — which fails the *entire* patch, not
// just that one field, silently discarding every other change in the same
// request (status, progress, subtasks, etc).
func camelKeysToSnake(patch map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(patch))
	for k, v := range patch {
		out[camelToSnake(k)] = v
	}
	return out
}