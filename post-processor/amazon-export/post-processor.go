//go:generate mapstructure-to-hcl2 -type Config

package amazonexport

import (
	"context"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/common"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	awscommon "github.com/hashicorp/packer/builder/amazon/common"
)

const BuilderId = "packer.post-processor.amazon-export"

// Configuration of this post processor
type Config struct {
	common.PackerConfig    `mapstructure:",squash"`
	awscommon.AccessConfig `mapstructure:",squash"`

	// Variables specific to this post processor
	ClientToken     string            `mapstructure:"client_token"`
	Description     string            `mapstructure:"description"`
	DiskImageFormat string            `mapstructure:"disk_image_format" required:"true"`
	DryRun          bool              `mapstructure:"dry_run"`
	ImageId         string            `mapstructure:"image_id" required:"true"`
	RoleName        string            `mapstructure:"role_name"`
	S3Bucket        string            `mapstructure:"s3_bucket_name" required:"true"`
	S3Prefix        string            `mapstructure:"s3_bucket_prefix"`
	Tags            map[string]string `mapstructure:"tags"`

	ctx interpolate.Context
}

type PostProcessor struct {
	config Config
}

func (p *PostProcessor) ConfigSpec() hcldec.ObjectSpec { return p.config.FlatMapstructure().HCL2Spec() }

func (p *PostProcessor) Configure(raws ...interface{}) error {
	return nil
}

func (p *PostProcessor) PostProcess(ctx context.Context, ui packersdk.Ui, artifact packersdk.Artifact) (packersdk.Artifact, bool, bool, error) {
	return &awscommon.Artifact{}, false, false, nil
}
