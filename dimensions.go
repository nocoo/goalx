package goalx

import "fmt"

// BuiltinDimensions are named goal dimensions for session guidance.
var BuiltinDimensions = map[string]string{
	"depth":         "Depth: Pick the single most impactful area and go as deep as possible. Trace code paths end-to-end. Prefer one thoroughly verified finding over five shallow ones.",
	"breadth":       "Breadth: Scan all dimensions to build a complete map. Cover every major component. Find blind spots and unexpected connections.",
	"creative":      "Creative: Think beyond conventional approaches. Propose non-obvious solutions. Challenge assumptions about what's possible. Look for elegant simplifications.",
	"feasibility":   "Feasibility: For every proposal, assess implementation cost, risk, dependencies, and timeline. Separate easy wins from heavy lifts. Be concrete about effort.",
	"adversarial":   "Adversarial: Your job is to find problems. Look for bugs, design flaws, edge cases, and incorrect assumptions. If something looks fine, try harder to break it.",
	"evidence":      "Evidence: Quantify everything. Run benchmarks, measure build times, count lines/functions/dependencies, check test coverage. No opinions without data.",
	"perfectionist": "Perfectionist: Demand ironclad evidence for every claim. Cite exact code references. Prefer fewer high-quality findings over many shallow ones. Re-read before commit. Depth over breadth.",
	"comparative":   "Comparative: Compare with industry best practices, similar projects, and established patterns. Identify where deviations are intentional strengths or accidental weaknesses.",
	"user":          "User perspective: Think from the end user's point of view. What's the experience like? What's confusing? What's missing? Focus on usability and developer ergonomics.",
}

// ResolveDimensions converts dimension names to session guidance strings using
// the built-in catalog plus any additional custom dimensions passed in.
func ResolveDimensions(names []string, custom ...map[string]string) ([]string, error) {
	return resolveDimensionsWithCatalog(BuiltinDimensions, names, custom...)
}

func resolveDimensionsWithCatalog(base map[string]string, names []string, custom ...map[string]string) ([]string, error) {
	merged := make(map[string]string, len(base))
	for k, v := range base {
		merged[k] = v
	}
	for _, m := range custom {
		for k, v := range m {
			merged[k] = v
		}
	}

	hints := make([]string, 0, len(names))
	for _, name := range names {
		hint, ok := merged[name]
		if !ok {
			return nil, fmt.Errorf("unknown dimension %q", name)
		}
		hints = append(hints, hint)
	}
	return hints, nil
}
