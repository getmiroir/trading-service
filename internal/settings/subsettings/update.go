package subsettings

import "go.yaml.in/yaml/v4"

// Update configures the desktop app update check. Read by the GUI only; the
// headless copy-trading service ignores it.
type Update struct {
	// URL of the release manifest listing the published desktop app versions.
	ManifestURL string `yaml:"manifest_url"`
	// Page opened when the user accepts the update prompt.
	DownloadURL string `yaml:"download_url"`
}

func DefaultUpdate() Update {
	return Update{
		ManifestURL: "https://getmiroir.xyz/manifest.json",
		DownloadURL: "https://getmiroir.xyz/download",
	}
}

func (u *Update) UnmarshalYAML(node *yaml.Node) error {
	*u = DefaultUpdate()
	type selfAlias Update
	return node.Decode((*selfAlias)(u))
}
