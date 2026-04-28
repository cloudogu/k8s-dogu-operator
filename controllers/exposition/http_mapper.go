package exposition

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccess"
	expv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
)

// buildHTTPEntries maps collected legacy routes to Exposition HTTP entries for a given Kubernetes Service.
func buildHTTPEntries(serviceName string, routes []serviceaccess.Route) ([]expv1.HTTPEntry, error) {
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

func buildHTTPEntry(serviceName string, route serviceaccess.Route) (expv1.HTTPEntry, error) {
	entry := expv1.HTTPEntry{
		Name:    fmt.Sprintf("%s-%d", route.Name, route.Port),
		Service: serviceName,
		Port:    route.Port,
		Path:    route.Location,
	}

	rewritePath, rewrite, err := getRewriteConfig(route)
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

type RouteRewrite struct {
	Pattern     string
	Replacement string
}

type serviceRewrite struct {
	Pattern string `json:"pattern"`
	Rewrite string `json:"rewrite"`
}

func getRewriteConfig(route serviceaccess.Route) (string, *RouteRewrite, error) {
	// An explicit rewrite takes precedence over the pass/location fallback.
	if route.Rewrite != "" {
		return parseRewrite(route.Rewrite)
	}

	location := strings.TrimSuffix(route.Location, "/")
	pass := strings.TrimSuffix(route.Pass, "/")
	// No rewrite is needed if the route already points to the same path.
	if route.Pass == "" || pass == location {
		return route.Location, nil, nil
	}

	// Otherwise rewrite requests from the exposed location to the target pass path.
	return route.Location, &RouteRewrite{
		Pattern:     "^" + regexp.QuoteMeta(route.Location) + "/?(.*)$",
		Replacement: route.Pass + "$1",
	}, nil
}

func parseRewrite(rawRewrite string) (string, *RouteRewrite, error) {
	rewrite := serviceRewrite{}
	if err := json.Unmarshal([]byte(strings.Trim(rawRewrite, "'")), &rewrite); err != nil {
		return "", nil, fmt.Errorf("failed to parse rewrite config: %w", err)
	}

	rewritePath := ensureLeadingSlash(rewrite.Pattern)
	replacementPrefix := ensureLeadingSlash(rewrite.Rewrite)

	if !strings.HasSuffix(replacementPrefix, "/") {
		replacementPrefix += "/"
	}

	return rewritePath, &RouteRewrite{
		Pattern:     "^" + regexp.QuoteMeta(rewritePath) + "(/|$)(.*)",
		Replacement: replacementPrefix + "$2",
	}, nil
}

func ensureLeadingSlash(value string) string {
	if strings.HasPrefix(value, "/") {
		return value
	}

	return "/" + value
}
