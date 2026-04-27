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
				Name:       "admin",
				Port:       80,
				Path:       "/admin",
				TargetPath: "/admin",
			},
		}

		entries := BuildHTTPEntries("cas", routes)

		assert.Equal(t, []expv1.HTTPEntry{
			{
				Name:    "admin",
				Service: "cas",
				Port:    80,
				Path:    "/admin",
			},
		}, entries)
	})

	t.Run("should map routes with regex rewrite if path and target path differ", func(t *testing.T) {
		routes := []Route{
			{
				Name:       "admin-api",
				Port:       80,
				Path:       "/api",
				TargetPath: "/admin/api/v2/",
			},
		}

		entries := BuildHTTPEntries("cas", routes)

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
				Name:       "admin",
				Port:       80,
				Path:       "/admin",
				TargetPath: "/admin",
			},
		}

		spec := BuildSpec("cas", routes)

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
				Name:       "admin",
				Port:       80,
				Path:       "/admin",
				TargetPath: "",
			},
		}

		entries := BuildHTTPEntries("cas", routes)

		assert.Nil(t, entries[0].Rewrite)
	})

	t.Run("should not create rewrite if paths only differ by trailing slash", func(t *testing.T) {
		routes := []Route{
			{
				Name:       "admin",
				Port:       80,
				Path:       "/admin/",
				TargetPath: "/admin",
			},
		}

		entries := BuildHTTPEntries("cas", routes)

		assert.Nil(t, entries[0].Rewrite)
	})
}

func TestSplitImagePortConfig(t *testing.T) {
	t.Run("should parse port without protocol", func(t *testing.T) {
		port, protocol, err := SplitImagePortConfig("8080")

		assert.NoError(t, err)
		assert.Equal(t, int32(8080), port)
		assert.Equal(t, "TCP", string(protocol))
	})

	t.Run("should parse port with protocol", func(t *testing.T) {
		port, protocol, err := SplitImagePortConfig("53/udp")

		assert.NoError(t, err)
		assert.Equal(t, int32(53), port)
		assert.Equal(t, "UDP", string(protocol))
	})

	t.Run("should fail for invalid port", func(t *testing.T) {
		_, _, err := SplitImagePortConfig("http/tcp")

		assert.Error(t, err)
		assert.ErrorContains(t, err, "error parsing int")
	})
}
