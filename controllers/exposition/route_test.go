package exposition

import (
	"testing"

	expv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestBuildHTTPEntries(t *testing.T) {
	t.Run("should map routes without rewrite if path and target path are equal", func(t *testing.T) {
		routes := []Route{
			{
				Name:     "admin",
				Port:     80,
				Location: "/admin",
				Pass:     "/admin",
			},
		}

		entries, err := BuildHTTPEntries("cas", routes)
		assert.NoError(t, err)
		assert.Equal(t, []expv1.HTTPEntry{
			{
				Name:    "admin",
				Service: "cas",
				Port:    80,
				Path:    "/admin",
			},
		}, entries)
	})

	t.Run("should map routes with normalized regex rewrite", func(t *testing.T) {
		routes := []Route{
			{
				Name:     "admin-api",
				Port:     80,
				Location: "/api",
				Pass:     "/admin/api/v2/",
			},
		}

		entries, err := BuildHTTPEntries("cas", routes)
		assert.NoError(t, err)
		assert.Equal(t, []expv1.HTTPEntry{
			{
				Name:    "admin-api",
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

	t.Run("should build exposition spec with http entries", func(t *testing.T) {
		routes := []Route{
			{
				Name:     "admin",
				Port:     80,
				Location: "/admin",
				Pass:     "/admin",
			},
		}

		spec, err := BuildSpec("cas", routes)
		assert.NoError(t, err)
		assert.Equal(t, expv1.ExpositionSpec{
			HTTP: []expv1.HTTPEntry{
				{
					Name:    "admin",
					Service: "cas",
					Port:    80,
					Path:    "/admin",
				},
			},
		}, spec)
	})

	t.Run("should not create rewrite if target path is empty", func(t *testing.T) {
		routes := []Route{
			{
				Name:     "admin",
				Port:     80,
				Location: "/admin",
				Pass:     "",
			},
		}

		entries, err := BuildHTTPEntries("cas", routes)
		assert.NoError(t, err)
		assert.Nil(t, entries[0].Rewrite)
	})

	t.Run("should not create rewrite if paths only differ by trailing slash", func(t *testing.T) {
		routes := []Route{
			{
				Name:     "admin",
				Port:     80,
				Location: "/admin/",
				Pass:     "/admin",
			},
		}

		entries, err := BuildHTTPEntries("cas", routes)
		assert.NoError(t, err)
		assert.Nil(t, entries[0].Rewrite)
	})

	t.Run("should map legacy rewrite path and regex", func(t *testing.T) {
		routes := []Route{
			{
				Name:     "admin",
				Port:     80,
				Location: "/admin",
				Pass:     "/admin",
				Rewrite:  `'{"pattern":"portainer","rewrite":""}'`,
			},
		}

		entries, err := BuildHTTPEntries("cas", routes)
		assert.NoError(t, err)
		assert.Equal(t, []expv1.HTTPEntry{
			{
				Name:    "admin",
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

	t.Run("should fail for invalid legacy rewrite", func(t *testing.T) {
		routes := []Route{
			{
				Name:     "admin",
				Port:     80,
				Location: "/admin",
				Pass:     "/admin",
				Rewrite:  `{"pattern":"broken"`,
			},
		}

		entries, err := BuildHTTPEntries("cas", routes)
		assert.Error(t, err)
		assert.Nil(t, entries)
		assert.ErrorContains(t, err, "failed to parse legacy rewrite config")
	})
}
