package exposition

import (
	"fmt"

	expv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
)

type Route struct {
	Name     string `json:"name"`
	Port     int32  `json:"port"`
	Location string `json:"location"`
	Pass     string `json:"pass"`
	Rewrite  string `json:"rewrite,omitempty"`
}

type RouteRewrite struct {
	Pattern     string
	Replacement string
}

// BuildHTTPEntries maps collected legacy routes to Exposition HTTP entries for a given Kubernetes Service.
func BuildHTTPEntries(serviceName string, routes []Route) ([]expv1.HTTPEntry, error) {
	entries := make([]expv1.HTTPEntry, 0, len(routes))

	for _, route := range routes {
		entry, err := buildHTTPEntry(serviceName, route)
		if err != nil {
			return nil, fmt.Errorf("failed to build HTTP entry for route %q: %w", route.Name, err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func buildHTTPEntry(serviceName string, route Route) (expv1.HTTPEntry, error) {
	entry := expv1.HTTPEntry{
		Name:    route.Name,
		Service: serviceName,
		Port:    route.Port,
		Path:    route.Location,
	}

	rewritePath, rewrite, err := normalizeRewrite(route)
	if err != nil {
		return expv1.HTTPEntry{}, err
	}

	if rewritePath != "" {
		entry.Path = rewritePath
	}

	if rewrite != nil {
		entry.Rewrite = &expv1.Rewrite{
			Regex: &expv1.RegexRewrite{
				Pattern:     rewrite.Pattern,
				Replacement: rewrite.Replacement,
			},
		}
	}

	return entry, nil
}

// BuildSpec maps collected legacy routes to an Exposition HTTP spec for a given Kubernetes Service.
func BuildSpec(serviceName string, routes []Route) (expv1.ExpositionSpec, error) {
	httpEntries, err := BuildHTTPEntries(serviceName, routes)
	if err != nil {
		return expv1.ExpositionSpec{}, err
	}

	return expv1.ExpositionSpec{
		HTTP: httpEntries,
	}, nil
}
