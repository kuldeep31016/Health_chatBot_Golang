package tools

import "strings"

func GetHealthData(query string) map[string]interface{} {
	q := strings.ToLower(query)
	profile := RawProfile()
	if profile == nil {
		return map[string]interface{}{}
	}

	result := map[string]interface{}{}

	containsAny := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(q, w) {
				return true
			}
		}
		return false
	}

	if containsAny("heart", "hrv", "bp") {
		if v, ok := profile["cardiovascular_matrix"]; ok {
			result["cardiovascular_matrix"] = v
		}
	}

	if containsAny("glucose", "weak", "dizzy") {
		if v, ok := profile["metabolic"]; ok {
			result["metabolic"] = v
		}
		if v, ok := profile["biomarkers"]; ok {
			result["biomarkers"] = v
		}
	}

	if containsAny("exercise", "workout", "run") {
		if v, ok := profile["fitness_milestones"]; ok {
			result["fitness_milestones"] = v
		}
		if v, ok := profile["workout_preferences"]; ok {
			result["workout_preferences"] = v
		}
	}

	if containsAny("allergy") {
		if v, ok := profile["allergies"]; ok {
			result["allergies"] = v
		}
	}

	if containsAny("appointment") {
		if v, ok := profile["scheduled_appointments"]; ok {
			result["scheduled_appointments"] = v
		}
	}

	return result
}
