package v2

// Capability represents a single POSIX capability.
//
// See docs at https://manned.org/capabilities.7
type Capability string

// Capabilities represent POSIX capabilities that can be added to or removed from a dogu.
//
// The fields Add and Drop will modify the capabilities as provided by the dogu descriptor. Add will append
// further capabilities while Drop will remove capabilities. The capability All can be used to add or remove all
// available capabilities.
//
// If the dogu descriptor only allows Fowner and Chown, this example will result in the following capability list: Fowner, Syslog
//
//	"Capabilities": {
//	   "Drop": "Chown"
//	   "Add": "Syslog"
//	}
//
// This example will always result in the following capability list: NetBindService
//
//	"Capabilities": {
//	   "Drop": ["All"],
//	   "Add": ["NetBindService", "Kill"]
//	}
type Capabilities struct {
	// Add contains the capabilities that should be allowed to be used in a container. This list is optional.
	// +optional
	// +listType=atomic
	Add []Capability `json:"add,omitempty"`
	// Drop contains the capabilities that should be blocked from being used in a container. This list is optional.
	// +optional
	// +listType=atomic
	Drop []Capability `json:"drop,omitempty"`
}

// SELinuxOptions are the labels to be applied to the container
type SELinuxOptions struct {
	// User is a SELinux user label that applies to the container.
	// +optional
	User string `json:"user,omitempty"`
	// Role is a SELinux role label that applies to the container.
	// +optional
	Role string `json:"role,omitempty"`
	// Type is a SELinux type label that applies to the container.
	// +optional
	Type string `json:"type,omitempty"`
	// Level is SELinux level label that applies to the container.
	// +optional
	Level string `json:"level,omitempty"`
}

// SeccompProfile defines a pod/container's seccomp profile settings.
// Only one profile source may be set.
// +union
type SeccompProfile struct {
	// type indicates which kind of seccomp profile will be applied.
	// Valid options are:
	//
	// Localhost - a profile defined in a file on the node should be used.
	// RuntimeDefault - the container runtime default profile should be used.
	// Unconfined - no profile should be applied.
	// +unionDiscriminator
	Type SeccompProfileType `json:"type"`
	// localhostProfile indicates a profile defined in a file on the node should be used.
	// The profile must be preconfigured on the node to work.
	// Must be a descending path, relative to the kubelet's configured seccomp profile location.
	// Must be set if type is "Localhost". Must NOT be set for any other type.
	// +optional
	LocalhostProfile *string `json:"localhostProfile,omitempty"`
}

// SeccompProfileType defines the supported seccomp profile types.
// +enum
type SeccompProfileType string

const (
	// SeccompProfileTypeUnconfined indicates no seccomp profile is applied (A.K.A. unconfined).
	SeccompProfileTypeUnconfined SeccompProfileType = "Unconfined"
	// SeccompProfileTypeRuntimeDefault represents the default container runtime seccomp profile.
	SeccompProfileTypeRuntimeDefault SeccompProfileType = "RuntimeDefault"
	// SeccompProfileTypeLocalhost indicates a profile defined in a file on the node should be used.
	// The file's location relative to <kubelet-root-dir>/seccomp.
	SeccompProfileTypeLocalhost SeccompProfileType = "Localhost"
)

// AppArmorProfile defines a pod or container's AppArmor settings.
// +union
type AppArmorProfile struct {
	// type indicates which kind of AppArmor profile will be applied.
	// Valid options are:
	//   Localhost - a profile pre-loaded on the node.
	//   RuntimeDefault - the container runtime's default profile.
	//   Unconfined - no AppArmor enforcement.
	// +unionDiscriminator
	Type AppArmorProfileType `json:"type"`

	// localhostProfile indicates a profile loaded on the node that should be used.
	// The profile must be preconfigured on the node to work.
	// Must match the loaded name of the profile.
	// Must be set if and only if type is "Localhost".
	// +optional
	LocalhostProfile *string `json:"localhostProfile,omitempty"`
}

// AppArmorProfileType references which type of AppArmor profile should be used.
// +enum
type AppArmorProfileType string

const (
	// AppArmorProfileTypeUnconfined indicates that no AppArmor profile should be enforced.
	AppArmorProfileTypeUnconfined AppArmorProfileType = "Unconfined"
	// AppArmorProfileTypeRuntimeDefault indicates that the container runtime's default AppArmor
	// profile should be used.
	AppArmorProfileTypeRuntimeDefault AppArmorProfileType = "RuntimeDefault"
	// AppArmorProfileTypeLocalhost indicates that a profile pre-loaded on the node should be used.
	AppArmorProfileTypeLocalhost AppArmorProfileType = "Localhost"
)

// Security overrides security policies defined in the dogu descriptor.
// These fields can be used to further reduce a dogu's attack surface.
type Security struct {
	// Capabilities sets the allowed and dropped capabilities for the dogu. The dogu should not use more than the
	// configured capabilities here, otherwise failure may occur at start-up or at run-time.
	// +optional
	Capabilities Capabilities `json:"capabilities,omitempty"`
	// RunAsNonRoot indicates that the container must run as a non-root user. The dogu must support running as non-root
	// user otherwise the dogu start may fail. This flag is optional and defaults to nil.
	// If nil, the value defined in the dogu descriptor is used.
	// +optional
	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty"`
	// ReadOnlyRootFileSystem mounts the container's root filesystem as read-only. The dogu must support accessing the
	// root file system by only reading otherwise the dogu start may fail. This flag is optional and defaults to nil.
	// If nil, the value defined in the dogu descriptor is used.
	// +optional
	ReadOnlyRootFileSystem *bool `json:"readOnlyRootFileSystem,omitempty"`
	// The SELinux context to be applied to the container.
	// If unspecified, the container runtime will allocate a random SELinux context for each
	// container, which is kubernetes default behaviour.
	// +optional
	SELinuxOptions *SELinuxOptions `json:"seLinuxOptions,omitempty"`
	// The seccomp options to use by this container.
	// +optional
	SeccompProfile *SeccompProfile `json:"seccompProfile,omitempty"`
	// appArmorProfile is the AppArmor options to use by this container.
	// +optional
	AppArmorProfile *AppArmorProfile `json:"appArmorProfile,omitempty"`
}
