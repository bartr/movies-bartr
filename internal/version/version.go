package version

// Version is the running build's semver. Override at build time with:
//   -ldflags "-X github.com/bartr/bartr-movies/internal/version.Version=X.Y.Z"
var Version = "0.1.0"
