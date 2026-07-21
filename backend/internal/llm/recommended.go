package llm

import "strings"

// Recommended picks first catalog match from preference lists (ASUTPORT roles).
func Recommended(catalog []string) map[string]string {
	set := map[string]struct{}{}
	for _, id := range catalog {
		set[id] = struct{}{}
	}
	pick := func(prefs []string) string {
		for _, p := range prefs {
			if _, ok := set[p]; ok {
				return p
			}
		}
		for _, p := range prefs {
			for id := range set {
				if id == p || strings.HasPrefix(id, p+"-") || strings.HasPrefix(id, p+":") {
					return id
				}
			}
		}
		return ""
	}
	out := map[string]string{}
	if v := pick([]string{
		"openai/text-embedding-3-large",
		"text-embedding-3-large",
	}); v != "" {
		out["embed_model"] = v
	}
	if v := pick([]string{
		"google/gemini-2.5-flash",
		"google/gemini-2.0-flash-001",
		"openai/gpt-4o-mini",
	}); v != "" {
		out["qualify_model"] = v
	}
	if v := pick([]string{
		"anthropic/claude-sonnet-4",
		"anthropic/claude-3.5-sonnet",
		"openai/gpt-4o",
	}); v != "" {
		out["answer_model"] = v
		out["kb_model"] = v
	}
	if v := pick([]string{
		"google/gemini-2.5-pro",
		"google/gemini-2.5-pro-preview",
		"openai/gpt-4o",
	}); v != "" {
		out["vision_model"] = v
	}
	return out
}
