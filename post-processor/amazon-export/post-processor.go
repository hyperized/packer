//go:generate mapstructure-to-hcl2 -type Config

package amazonexport

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/common"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	awscommon "github.com/hashicorp/packer/builder/amazon/common"
	"log"
	"strings"
)

const BuilderId = "packer.post-processor.amazon-export"
const AwsExportAccount = "vm-import-export@amazon.com"
const DefaultDescription = "packer-export-{{timestamp}}"
const DefaultRoleName = "vmimport"

// Configuration of this post processor
type Config struct {
	common.PackerConfig    `mapstructure:",squash"`
	awscommon.AccessConfig `mapstructure:",squash"`

	// Variables specific to this post processor
	ClientToken     string            `mapstructure:"client_token"`
	Description     string            `mapstructure:"description"`
	DiskImageFormat string            `mapstructure:"disk_image_format" required:"true"`
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

func (p *PostProcessor) ConfigSpec() hcldec.ObjectSpec {
	return p.config.FlatMapstructure().HCL2Spec()
}

func (p *PostProcessor) Configure(raws ...interface{}) error {
	var (
		supportedDiskFormats = map[string]bool{
			"VMDK": true,
			"RAW":  true,
			"VHD":  true,
		}
		requiredParameters = map[string]*string{
			"image_id":       &p.config.ImageId,
			"s3_bucket_name": &p.config.S3Bucket,
		}
		err = config.Decode(&p.config, &config.DecodeOpts{
			PluginType:         BuilderId,
			Interpolate:        true,
			InterpolateContext: &p.config.ctx,
			InterpolateFilter:  &interpolate.RenderFilter{},
		}, raws...)
		errs = new(packersdk.MultiError)
	)
	// TODO: understand if Interpolation is required

	// If config cannot be decoded, don't bother with the rest.
	if err != nil {
		return err
	}

	// Slipstream TemplateFuncs
	p.config.ctx.Funcs = awscommon.TemplateFuncs

	// Set defaults
	if p.config.Description == "" {
		p.config.Description = DefaultDescription
	}

	if p.config.RoleName == "" {
		p.config.RoleName = DefaultRoleName
	}

	// AWS documentation specifies upper-case strings.
	p.config.DiskImageFormat = strings.ToUpper(p.config.DiskImageFormat)

	// Validate disk formats
	if !supportedDiskFormats[p.config.DiskImageFormat] {
		errs = packersdk.MultiErrorAppend(
			errs, fmt.Errorf(
				"invalid disk image format. Only 'VMDK', 'RAW', or 'VHD' are allowed",
				p.config.DiskImageFormat,
			),
		)
	}

	// Validate if parameters are in fact not empty strings
	for key, value := range requiredParameters {
		if !(len(*value) > 0) {
			errs = packersdk.MultiErrorAppend(
				errs,
				fmt.Errorf("no value provided for image_id", key),
			)
		}
	}

	// TODO: Validate if bucket exists and is writable for Amazon specific user.

	// Send it
	if len(errs.Errors) > 0 {
		return errs
	}

	// Configure polling
	if p.config.PollingConfig == nil {
		p.config.PollingConfig = new(awscommon.AWSPollingConfig)
	}
	p.config.PollingConfig.LogEnvOverrideWarnings()

	// Ensure we filter secrets out from log and then log
	packersdk.LogSecretFilter.Set(p.config.AccessKey, p.config.SecretKey, p.config.Token)
	log.Println(p.config)

	return nil
}

func (p *PostProcessor) PostProcess(ctx context.Context, ui packersdk.Ui, artifact packersdk.Artifact) (packersdk.Artifact, bool, bool, error) {
	var (
		session, sessionError = p.config.Session()
		ec2Client             = ec2.New(session)
		ExportImageInput      = ec2.ExportImageInput{
			ClientToken:     &p.config.ClientToken,
			Description:     &p.config.Description,
			DiskImageFormat: &p.config.DiskImageFormat,
			DryRun:          &[]bool{false}[0],
			ImageId:         &p.config.ImageId,
			RoleName:        &p.config.RoleName,
			S3ExportLocation: &ec2.ExportTaskS3LocationRequest{
				S3Bucket: &p.config.S3Bucket,
				S3Prefix: &p.config.S3Prefix,
			},
			TagSpecifications: nil,
		}
		exportImageOutput, exportError = ec2Client.ExportImageWithContext(ctx, &ExportImageInput)
		waitError                      = p.config.PollingConfig.WaitUntilImageExported(aws.BackgroundContext(), ec2Client, exportImageOutput.ExportImageTaskId)
	)

	if sessionError != nil {
		return nil, false, false, sessionError
	}

	if exportError != nil {
		return nil, false, false, exportError
	}

	if waitError != nil {
		return nil, false, false, waitError
	}

	// TODO: possibly re-fetch image export results to obtain more detailed error messages

	// TODO: Replace artifact with our own, as common artifact infact means AMI, not S3 resource
	return &awscommon.Artifact{}, false, false, nil
}
