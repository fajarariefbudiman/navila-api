package handlers

import (
	"strings"

	"goals-api/models"
)

// computeProgressFromSubtasks menghitung persentase progress (0-100)
// berdasarkan berapa subtask yang sudah dicentang. Dipakai bareng oleh
// goal_handler.go & plan_handler.go supaya progress SELALU sinkron
// dengan checklist -- user gak perlu nebak-nebak sendiri sudah di titik
// mana, karena progress otomatis nambah begitu subtask dicentang.
//
// Kalau subtasks kosong (goal/plan tanpa checklist), return -1 sebagai
// penanda "jangan diubah" -- progress-nya tetap dikontrol manual oleh
// user (mis. lewat perubahan status).
func computeProgressFromSubtasks(subtasks models.Subtasks) int {
	if len(subtasks) == 0 {
		return -1
	}
	done := 0
	for _, s := range subtasks {
		if s.Done {
			done++
		}
	}
	return (done * 100) / len(subtasks)
}

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
func camelKeysToSnake(patch map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(patch))
	for k, v := range patch {
		out[camelToSnake(k)] = v
	}
	return out
}