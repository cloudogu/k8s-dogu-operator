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
	Add []Capability `json:"add,omitempty"`
	// Drop contains the capabilities that should be blocked from being used in a container. This list is optional.
	Drop []Capability `json:"drop,omitempty"`
}

// Security overrides security policies defined in the dogu descriptor. These fields can be used to further reduce a dogu's attack surface.
//
// Example:
//
//	"Security": {
//	  "Capabilities": {
//	     "Drop": ["All"],
//	     "Add": ["NetBindService", "Kill"]
//	   },
//	  "RunAsNonRoot": true,
//	  "ReadOnlyRootFileSystem": true
//	}
type Security struct {
	// Capabilities sets the allowed and dropped capabilities for the dogu. The dogu should not use more than the
	// configured capabilities here, otherwise failure may occur at start-up or at run-time. This list is optional.
	Capabilities Capabilities `json:"capabilities,omitempty"`
	// RunAsNonRoot indicates that the container must run as a non-root user. The dogu must support running as non-root
	// user otherwise the dogu start may fail. This flag is optional and defaults to nil.
	// If nil, the value defined in the dogu descriptor is used.
	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty"`
	// ReadOnlyRootFileSystem mounts the container's root filesystem as read-only. The dogu must support accessing the
	// root file system by only reading otherwise the dogu start may fail. This flag is optional and defaults to nil.
	// If nil, the value defined in the dogu descriptor is used.
	ReadOnlyRootFileSystem *bool `json:"readOnlyRootFileSystem,omitempty"`
}
