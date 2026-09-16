package orientation

func combineCheckStatus(current, incoming string) string {
	if current == "fail" || incoming == "fail" {
		return "fail"
	}
	if current == "unknown" || incoming == "unknown" {
		return "unknown"
	}
	return "pass"
}

func mergeChecks(result *Check, checks ...Check) {
	seen := map[Finding]bool{}
	for _, reason := range result.Reasons {
		seen[reason] = true
	}
	for _, check := range checks {
		result.Status = combineCheckStatus(result.Status, check.Status)
		for _, reason := range check.Reasons {
			if !seen[reason] {
				result.Reasons = append(result.Reasons, reason)
				seen[reason] = true
			}
		}
	}
}
