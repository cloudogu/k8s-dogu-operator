package exposition

import (
	"regexp"
	"strings"

	expv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
)

type Route struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	Path       string `json:"location"`
	TargetPath string `json:"pass"`
	Rewrite    string `json:"rewrite,omitempty"`
}

// BuildHTTPEntries maps web routes to Exposition HTTP entries for a given Kubernetes Service.
func BuildHTTPEntries(serviceName string, routes []Route) []expv1.HTTPEntry {
	entries := make([]expv1.HTTPEntry, 0, len(routes))

	for _, route := range routes {
		entry := expv1.HTTPEntry{
			Name:    route.Name,
			Service: serviceName,
			Port:    route.Port,
			Path:    route.Path,
		}

		if rewrite := buildRewrite(route); rewrite != nil {
			entry.Rewrite = rewrite
		}

		entries = append(entries, entry)
	}

	return entries
}

func buildRewrite(route Route) *expv1.Rewrite {
	if route.TargetPath == "" || route.TargetPath == route.Path {
		return nil
	}

	normalizedLocation := strings.TrimSuffix(route.Path, "/")
	normalizedPass := strings.TrimSuffix(route.TargetPath, "/")
	if normalizedPass == normalizedLocation {
		return nil
	}

	pattern := "^" + regexp.QuoteMeta(route.Path) + "/?(.*)$"
	replacement := route.TargetPath + "$1"

	return &expv1.Rewrite{
		Regex: &expv1.RegexRewrite{
			Pattern:     pattern,
			Replacement: replacement,
		},
	}
}

// BuildSpec maps web routes to an Exposition HTTP spec for a given Kubernetes Service.
func BuildSpec(serviceName string, routes []Route) expv1.ExpositionSpec {
	return expv1.ExpositionSpec{
		HTTP: BuildHTTPEntries(serviceName, routes),
	}
}
