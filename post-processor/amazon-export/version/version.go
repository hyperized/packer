package version

import (
	"github.com/hashicorp/packer-plugin-sdk/version"
	packerVersion "github.com/hashicorp/packer/version"
)

var AmazonExportPluginVersion *version.PluginVersion

func init() {
	AmazonExportPluginVersion = version.InitializePluginVersion(
		packerVersion.Version, packerVersion.VersionPrerelease)
}
