package modelsdev

// Thinking level keys Magic/Pi expose in thinkingLevelMap.
var thinkingLevels = []string{"minimal", "low", "medium", "high", "xhigh", "max"}

// EffortThinkingLevelMap ports Pi getEffortThinkingLevelMap:
// for each Pi level, if present in effort values → map to itself; else null.
// off → "none" if "none" is supported, else null.
// Returns nil when there are no usable effort values (toggle/budget alone do not produce a map).
func EffortThinkingLevelMap(options []ReasoningOption) map[string]*string {
	var effortValues []string
	for _, opt := range options {
		if opt.Type != "effort" {
			continue
		}
		for _, v := range opt.Values {
			if v == "" || v == "default" {
				continue
			}
			effortValues = append(effortValues, v)
		}
	}
	if len(effortValues) == 0 {
		return nil
	}
	supported := make(map[string]bool, len(effortValues))
	for _, v := range effortValues {
		supported[v] = true
	}
	hasLevel := supported["none"]
	for _, level := range thinkingLevels {
		if supported[level] {
			hasLevel = true
			break
		}
	}
	if !hasLevel {
		return nil
	}

	strPtr := func(s string) *string { return &s }
	out := make(map[string]*string, 1+len(thinkingLevels))
	if supported["none"] {
		out["off"] = strPtr("none")
	} else {
		out["off"] = nil
	}
	for _, level := range thinkingLevels {
		if supported[level] {
			out[level] = strPtr(level)
		} else {
			out[level] = nil
		}
	}
	return out
}

// OpenRouterThinkingLevelMap ports Pi getOpenRouterThinkingLevelMap.
// mandatory=true forces off: null (cannot disable).
func OpenRouterThinkingLevelMap(supportedEfforts []string, mandatory bool) map[string]*string {
	if len(supportedEfforts) == 0 {
		if mandatory {
			return map[string]*string{"off": nil}
		}
		return nil
	}
	opts := []ReasoningOption{{Type: "effort", Values: supportedEfforts}}
	m := EffortThinkingLevelMap(opts)
	if m == nil {
		if mandatory {
			return map[string]*string{"off": nil}
		}
		return nil
	}
	if mandatory {
		m["off"] = nil
	} else {
		none := "none"
		m["off"] = &none
	}
	return m
}
