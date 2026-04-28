package exposition

import (
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccess"
	expv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestBuildHTTPEntries(t *testing.T) {
	t.Run("should map routes without rewrite if path and target path are equal", func(t *testing.T) {
		routes := []serviceaccess.Route{
			{
				Name:     "admin",
				Port:     80,
				Location: "/admin",
				Pass:     "/admin",
			},
		}

		entries, err := buildHTTPEntries("cas", routes)
		assert.NoError(t, err)
		assert.Equal(t, []expv1.HTTPEntry{
			{
				Name:    "admin-80",
				Service: "cas",
				Port:    80,
				Path:    "/admin",
			},
		}, entries)
	})

	t.Run("should map routes with normalized regex rewrite", func(t *testing.T) {
		routes := []serviceaccess.Route{
			{
				Name:     "admin-api",
				Port:     80,
				Location: "/api",
				Pass:     "/admin/api/v2/",
			},
		}

		entries, err := buildHTTPEntries("cas", routes)
		assert.NoError(t, err)
		assert.Equal(t, []expv1.HTTPEntry{
			{
				Name:    "admin-api-80",
				Service: "cas",
				Port:    80,
				Path:    "/api",
				Rewrite: &expv1.Rewrite{
					Regex: &expv1.RegexRewrite{
						Pattern:     "^/api/?(.*)$",
						Replacement: "/admin/api/v2/$1",
					},
				},
			},
		}, entries)
	})

	t.Run("should not create rewrite if target path is empty", func(t *testing.T) {
		routes := []serviceaccess.Route{
			{
				Name:     "admin",
				Port:     80,
				Location: "/admin",
				Pass:     "",
			},
		}

		entries, err := buildHTTPEntries("cas", routes)
		assert.NoError(t, err)
		assert.Nil(t, entries[0].Rewrite)
	})

	t.Run("should not create rewrite if paths only differ by trailing slash", func(t *testing.T) {
		routes := []serviceaccess.Route{
			{
				Name:     "admin",
				Port:     80,
				Location: "/admin/",
				Pass:     "/admin",
			},
		}

		entries, err := buildHTTPEntries("cas", routes)
		assert.NoError(t, err)
		assert.Nil(t, entries[0].Rewrite)
	})

	t.Run("should map rewrite path and regex", func(t *testing.T) {
		routes := []serviceaccess.Route{
			{
				Name:     "admin",
				Port:     80,
				Location: "/admin",
				Pass:     "/admin",
				Rewrite:  `'{"pattern":"portainer","rewrite":""}'`,
			},
		}

		entries, err := buildHTTPEntries("cas", routes)
		assert.NoError(t, err)
		assert.Equal(t, []expv1.HTTPEntry{
			{
				Name:    "admin-80",
				Service: "cas",
				Port:    80,
				Path:    "/portainer",
				Rewrite: &expv1.Rewrite{
					Regex: &expv1.RegexRewrite{
						Pattern:     "^/portainer(/|$)(.*)",
						Replacement: "/$2",
					},
				},
			},
		}, entries)
	})

	t.Run("should fail for invalid rewrite", func(t *testing.T) {
		routes := []serviceaccess.Route{
			{
				Name:     "admin",
				Port:     80,
				Location: "/admin",
				Pass:     "/admin",
				Rewrite:  `{"pattern":"broken"`,
			},
		}

		entries, err := buildHTTPEntries("cas", routes)
		assert.Error(t, err)
		assert.Nil(t, entries)
		assert.ErrorContains(t, err, "failed to parse rewrite config")
	})

	t.Run("should create unique names for duplicate route names", func(t *testing.T) {
		routes := []serviceaccess.Route{
			{
				Name:     "jenkins",
				Port:     8080,
				Location: "/jenkins",
				Pass:     "/jenkins",
			},
			{
				Name:     "jenkins",
				Port:     50000,
				Location: "/jenkins-agent",
				Pass:     "/jenkins-agent",
			},
		}

		entries, err := buildHTTPEntries("jenkins", routes)
		assert.NoError(t, err)
		assert.Equal(t, []expv1.HTTPEntry{
			{
				Name:    "jenkins-8080",
				Service: "jenkins",
				Port:    8080,
				Path:    "/jenkins",
			},
			{
				Name:    "jenkins-50000",
				Service: "jenkins",
				Port:    50000,
				Path:    "/jenkins-agent",
			},
		}, entries)
	})
}
