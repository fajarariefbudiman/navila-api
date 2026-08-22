package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"goals-api/models"
)

// generatedEntry: bentuk mentah satu blok jadwal sebelum di-attach ID/UserID.
type generatedEntry struct {
	DayOfWeek int    `json:"dayOfWeek"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Title     string `json:"title"`
	Category  string `json:"category"`
}

// generateSchedule adalah pintu masuk tunggal: coba AI dulu kalau API key
// tersedia, kalau gagal/tidak ada key, otomatis fallback ke generator
// statis berbasis aturan supaya fitur tetap jalan tanpa AI.
func generateSchedule(apiKey string, p models.HabitProfile) []generatedEntry {
	if apiKey != "" {
		if entries, err := generateScheduleAI(apiKey, p); err == nil && len(entries) > 0 {
			return entries
		}
	}
	return generateScheduleStatic(p)
}

// ---------------------------------------------------------------------
// AI generator: memanggil Claude API (model kecil, cukup buat tugas
// terstruktur seperti ini) dan minta jawaban JSON array murni supaya
// gampang di-parse langsung jadi []generatedEntry.
// ---------------------------------------------------------------------
func generateScheduleAI(apiKey string, p models.HabitProfile) ([]generatedEntry, error) {
	prompt := fmt.Sprintf(`Buatkan jadwal mingguan (Senin=1 s.d Minggu=7) untuk satu orang dengan profil berikut:
- Gaji per bulan: %.0f
- Agama: %s
- Budget kontrakan/kos per bulan: %.0f
- Budget bensin per bulan: %.0f
- Budget makan per bulan: %.0f
- Sedang kuliah: %t
- Freelance: %t, jam %s - %s

Buat jadwal harian yang realistis (waktu ibadah sesuai agama, waktu kuliah kalau relevan, jam kerja/freelance, jam istirahat, dan reminder pengaturan keuangan mingguan). Balas HANYA dengan JSON array valid, tanpa teks lain, tanpa markdown, dengan format setiap elemen:
{"dayOfWeek": 1-7, "startTime": "HH:MM", "endTime": "HH:MM", "title": "...", "category": "ibadah|kuliah|kerja|freelance|istirahat|keuangan|lainnya"}`,
		p.Salary, p.Religion, p.RentBudget, p.FuelBudget, p.FoodBudget,
		p.IsCollege, p.HasFreelance, safeStr(p.FreelanceStart), safeStr(p.FreelanceEnd))

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":      "claude-sonnet-4-6",
		"max_tokens": 4000,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic api status %d", resp.StatusCode)
	}

	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	var text string
	for _, block := range parsed.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	var entries []generatedEntry
	if err := json.Unmarshal([]byte(text), &entries); err != nil {
		return nil, fmt.Errorf("failed to parse AI schedule: %w", err)
	}
	return entries, nil
}

func safeStr(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}

// ---------------------------------------------------------------------
// Static generator: fallback berbasis aturan sederhana kalau AI tidak
// tersedia. Tetap menghasilkan jadwal Senin-Minggu yang masuk akal.
// ---------------------------------------------------------------------
func generateScheduleStatic(p models.HabitProfile) []generatedEntry {
	var out []generatedEntry
	prayerTimes := []struct{ start, end, title string }{
		{"04:30", "05:00", "Sholat Subuh"},
		{"12:00", "12:30", "Sholat Dzuhur"},
		{"15:15", "15:45", "Sholat Ashar"},
		{"18:00", "18:30", "Sholat Maghrib"},
		{"19:15", "19:45", "Sholat Isya"},
	}

	for day := 1; day <= 7; day++ {
		isWeekend := day == 6 || day == 7

		if p.Religion == "islam" {
			for _, pt := range prayerTimes {
				out = append(out, generatedEntry{day, pt.start, pt.end, pt.title, "ibadah"})
			}
		} else if p.Religion == "kristen" || p.Religion == "katolik" {
			if day == 7 {
				out = append(out, generatedEntry{day, "08:00", "10:00", "Ibadah Minggu", "ibadah"})
			} else {
				out = append(out, generatedEntry{day, "05:30", "06:00", "Doa Pagi", "ibadah"})
			}
		} else if p.Religion == "hindu" {
			out = append(out, generatedEntry{day, "06:00", "06:30", "Persembahyangan Pagi", "ibadah"})
		} else if p.Religion == "buddha" {
			out = append(out, generatedEntry{day, "06:00", "06:30", "Meditasi Pagi", "ibadah"})
		}

		if p.IsCollege && !isWeekend {
			out = append(out, generatedEntry{day, "08:00", "12:00", "Kuliah", "kuliah"})
		}

		if !isWeekend {
			out = append(out, generatedEntry{day, "07:30", "08:00", "Sarapan", "istirahat"})
			startWork := "09:00"
			endWork := "17:00"
			if p.IsCollege {
				startWork = "13:00"
				endWork = "17:00"
			}
			out = append(out, generatedEntry{day, startWork, endWork, "Jam Kerja", "kerja"})
		}

		if p.HasFreelance && p.FreelanceStart != nil && p.FreelanceEnd != nil {
			out = append(out, generatedEntry{day, *p.FreelanceStart, *p.FreelanceEnd, "Freelance", "freelance"})
		}

		out = append(out, generatedEntry{day, "12:30", "13:00", "Makan Siang", "istirahat"})
		out = append(out, generatedEntry{day, "20:00", "20:30", "Makan Malam", "istirahat"})

		if day == 7 {
			out = append(out, generatedEntry{day, "16:00", "16:30", "Review Keuangan Mingguan", "keuangan"})
		}
	}
	return out
}
