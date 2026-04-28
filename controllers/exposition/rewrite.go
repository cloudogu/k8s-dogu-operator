package exposition

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

func normalizeRewrite(route Route) (string, *RouteRewrite, error) {
	if route.Rewrite != "" {
		rewritePath, rewrite, err := parseRewrite(route.Rewrite)
		if err != nil {
			return "", nil, err
		}

		return rewritePath, rewrite, nil
	}

	normalizedLocation := strings.TrimSuffix(route.Location, "/")
	normalizedPass := strings.TrimSuffix(route.Pass, "/")
	if route.Pass == "" || normalizedPass == normalizedLocation {
		return route.Location, nil, nil
	}

	return route.Location, &RouteRewrite{
		Pattern:     "^" + regexp.QuoteMeta(route.Location) + "/?(.*)$",
		Replacement: route.Pass + "$1",
	}, nil
}

func parseRewrite(rawRewrite string) (string, *RouteRewrite, error) {
	rewrite := legacyServiceRewrite{}
	if err := json.Unmarshal([]byte(strings.Trim(rawRewrite, "'")), &rewrite); err != nil {
		return "", nil, fmt.Errorf("failed to parse legacy rewrite config: %w", err)
	}

	rewritePath := ensureLeadingSlash(rewrite.Pattern)
	replacementPrefix := rewrite.Rewrite
	if replacementPrefix == "" {
		replacementPrefix = "/"
	} else {
		replacementPrefix = ensureLeadingSlash(replacementPrefix)
	}

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
