package tools

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func TryAnswerProfileQuestion(query string) (string, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return "", false
	}

	profile := RawProfile()
	if profile == nil {
		return "", false
	}

	if isAny(q, "age", "how old") {
		if v, ok := getNumber(profile, "age"); ok {
			return fmt.Sprintf("You are %.0f years old.", v), true
		}
	}

	if isAny(q, "today's date", "todays date", "what is the date", "date today", "current date") {
		return fmt.Sprintf("Today's date is %s.", time.Now().Format("2006-01-02")), true
	}

	if isAny(q, "weight", "weigh") {
		if v, ok := getNumber(profile, "weight_kg"); ok {
			return fmt.Sprintf("You weigh %.0f kg.", v), true
		}
	}

	if isAny(q, "height", "tall") {
		if v, ok := getNumber(profile, "height_cm"); ok {
			return fmt.Sprintf("Your height is %.0f cm.", v), true
		}
	}

	if isAny(q, "bmi") {
		weight, wOK := getNumber(profile, "weight_kg")
		heightCM, hOK := getNumber(profile, "height_cm")
		if wOK && hOK && heightCM > 0 {
			hM := heightCM / 100.0
			bmi := weight / (hM * hM)
			return fmt.Sprintf("Your BMI is %.1f based on %.0f kg and %.0f cm.", math.Round(bmi*10)/10, weight, heightCM), true
		}
	}

	if isAny(q, "name", "who am i") {
		if v, ok := getString(profile, "name"); ok {
			return fmt.Sprintf("Your name is %s.", v), true
		}
	}

	if isAny(q, "blood group", "blood type") {
		if v, ok := getString(profile, "blood_group"); ok {
			return fmt.Sprintf("Your blood group is %s.", v), true
		}
	}

	if isAny(q, "gender", "sex") {
		if v, ok := getString(profile, "gender"); ok {
			return fmt.Sprintf("Your gender is %s.", v), true
		}
	}

	if hasAllTerms(q, "hair", "color") || hasAllTerms(q, "hair", "colour") {
		if v, ok := getString(profile, "hair_color"); ok {
			return fmt.Sprintf("Your hair color is %s.", v), true
		}
	}

	if hasAllTerms(q, "eye", "color") || hasAllTerms(q, "eyes", "color") || hasAllTerms(q, "eye", "colour") || hasAllTerms(q, "eyes", "colour") {
		if v, ok := getString(profile, "eye_color"); ok {
			return fmt.Sprintf("Your eye color is %s.", v), true
		}
	}

	if isAny(q, "hemoglobin", "haemoglobin", "hb") {
		if biomarkers, ok := profile["biomarkers"].(map[string]interface{}); ok {
			if v, ok := getNumber(biomarkers, "hemoglobin_g_dl"); ok {
				return fmt.Sprintf("Your hemoglobin level is %.1f g/dL.", math.Round(v*10)/10), true
			}
		}
	}

	return "", false
}

func isAny(q string, terms ...string) bool {
	for _, t := range terms {
		if strings.Contains(q, t) {
			return true
		}
	}
	return false
}

func hasAllTerms(q string, terms ...string) bool {
	for _, t := range terms {
		if !strings.Contains(q, t) {
			return false
		}
	}
	return true
}

func getString(profile map[string]interface{}, key string) (string, bool) {
	raw, ok := profile[key]
	if !ok || raw == nil {
		return "", false
	}
	v, ok := raw.(string)
	if !ok || strings.TrimSpace(v) == "" {
		return "", false
	}
	return v, true
}

func getNumber(profile map[string]interface{}, key string) (float64, bool) {
	raw, ok := profile[key]
	if !ok || raw == nil {
		return 0, false
	}

	switch v := raw.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}
