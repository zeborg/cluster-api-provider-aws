package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"sigs.k8s.io/cluster-api-provider-aws/ci/custom"
)

func main() {
	cleanup := flag.Bool("cleanup", false, "Cleanup CAPA AMIs with 'test-' prefix after image-builder finishes building AMIs")
	flag.Parse()

	AMIBuildConfigFilename := os.Getenv("AMI_BUILD_CONFIG_FILENAME")
	AMIBuildConfigDefaultsFilename := os.Getenv("AMI_BUILD_CONFIG_DEFAULTS")

	ami_regions := os.Getenv("AMI_BUILD_REGIONS")
	ownerID := os.Getenv("AWS_AMI_OWNER_ID")
	supportedOS := strings.Split(os.Getenv("AMI_BUILD_SUPPORTED_OS"), ",")

	dat, err := os.ReadFile(AMIBuildConfigFilename)
	custom.CheckError(err, "")
	currentAMIBuildConfig := new(custom.AMIBuildConfig)
	err = json.Unmarshal(dat, currentAMIBuildConfig)
	custom.CheckError(err, "")

	dat, err = os.ReadFile(AMIBuildConfigDefaultsFilename)
	custom.CheckError(err, "")
	defaultAMIBuildConfig := new(custom.AMIBuildConfigDefaults)
	err = json.Unmarshal(dat, defaultAMIBuildConfig)
	custom.CheckError(err, "")

	log.Println("Creating new session")
	mySession := session.Must(session.NewSession())
	log.Println("New session created successfully")

	allRegions := []string{"ap-south-1", "eu-west-3", "eu-west-2", "eu-west-1", "ap-northeast-2", "ap-northeast-1", "sa-east-1", "ca-central-1",
		"ap-southeast-1", "ap-southeast-2", "eu-central-1", "us-east-1", "us-east-2", "us-west-1", "us-west-2"}
	allOS := []string{"amazon-2", "centos-7", "flatcar-stable", "ubuntu-18.04", "ubuntu-20.04"}
	amiNamePrefixFormat := "capa-ami-%s-%s"
	DefaultAMIOwnerID := "570412231501"

	if ownerID == "" {
		ownerID = DefaultAMIOwnerID
	}

	for _, v := range currentAMIBuildConfig.K8sReleases {
		amiExists := false

		for _, r := range allRegions {
			log.Println("Info: Creating new instance of EC2 client with region", r)
			svc := ec2.New(mySession, aws.NewConfig().WithRegion(r))
			log.Println("Info: New instance of EC2 client with region", r, "created successfully ")

			log.Println("Info: Checking if AMI for Kubernetes", v, "exists in region", r)
			for _, os := range allOS {
				amiNamePrefix := fmt.Sprintf(amiNamePrefixFormat, os, strings.TrimPrefix(v, "v"))

				descImgInput := &ec2.DescribeImagesInput{
					Owners: []*string{&ownerID},
					Filters: []*ec2.Filter{
						{
							Name:   aws.String("owner-id"),
							Values: []*string{aws.String(ownerID)},
						},
						{
							Name:   aws.String("architecture"),
							Values: []*string{aws.String("x86_64")},
						},
						{
							Name:   aws.String("state"),
							Values: []*string{aws.String("available")},
						},
						{
							Name:   aws.String("virtualization-type"),
							Values: []*string{aws.String("hvm")},
						},
					},
				}
				l, _ := svc.DescribeImages(descImgInput)

				for _, img := range l.Images {
					if strings.HasPrefix(*img.Name, amiNamePrefix) {
						log.Println("Info: AMI for Kubernetes", v, "found in region", r)
						amiExists = true
						break
					}
				}
			}
			if amiExists {
				break
			}
		}

		if !amiExists {
			log.Printf("Info: Building AMI for Kubernetes %s.", v)
			kubernetes_semver := v
			kubernetes_rpm_version := strings.TrimPrefix(v, "v") + "-0"
			kubernetes_deb_version := strings.TrimPrefix(v, "v") + "-00"
			kubernetes_series := strings.Split(v, ".")[0] + "." + strings.Split(v, ".")[1]

			flagsK8s := fmt.Sprintf("-var=ami_regions=%s -var=kubernetes_series=%s -var=kubernetes_semver=%s -var=kubernetes_rpm_version=%s -var=kubernetes_deb_version=%s ", ami_regions, kubernetes_series, kubernetes_semver, kubernetes_rpm_version, kubernetes_deb_version)
			for k, v := range defaultAMIBuildConfig.Default {
				flagsK8s += fmt.Sprintf("-var=%s=%s ", k, v)
			}

			for _, os := range supportedOS {
				switch os {
				case "amazon-2":
					flags := flagsK8s
					for k, v := range defaultAMIBuildConfig.Amazon2 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}

					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					log.Println(fmt.Sprintf("Info: flags:  \"%s\"", flags))

					stdout, stderr, err := custom.Shell(fmt.Sprintf("cd image-builder/images/capi && PACKER_FLAGS=\"%s\" make build-ami-%s", flags, os))
					custom.CheckError(err, stderr)
					if stderr != "" {
						log.Fatalf("Error: %s", stderr)
					} else {
						log.Println(stdout)
					}
				case "centos-7":
					flags := flagsK8s
					for k, v := range defaultAMIBuildConfig.Centos7 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}

					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					log.Println(fmt.Sprintf("Info: flags:  \"%s\"", flags))

					stdout, stderr, err := custom.Shell(fmt.Sprintf("cd image-builder/images/capi && PACKER_FLAGS=\"%s\" make build-ami-%s", flags, os))
					custom.CheckError(err, stderr)
					if stderr != "" {
						log.Fatalf("Error: %s", stderr)
					} else {
						log.Println(stdout)
					}
				case "flatcar":
					flags := flagsK8s
					for k, v := range defaultAMIBuildConfig.Flatcar {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}

					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					log.Println(fmt.Sprintf("Info: flags:  \"%s\"", flags))

					stdout, stderr, err := custom.Shell(fmt.Sprintf("cd image-builder/images/capi && PACKER_FLAGS=\"%s\" make build-ami-%s", flags, os))
					custom.CheckError(err, stderr)
					if stderr != "" {
						log.Fatalf("Error: %s", stderr)
					} else {
						log.Println(stdout)
					}
				case "ubuntu-1804":
					flags := flagsK8s
					for k, v := range defaultAMIBuildConfig.Ubuntu1804 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}

					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					log.Println(fmt.Sprintf("Info: flags:  \"%s\"", flags))

					stdout, stderr, err := custom.Shell(fmt.Sprintf("cd image-builder/images/capi && PACKER_FLAGS=\"%s\" make build-ami-%s", flags, os))
					custom.CheckError(err, stderr)
					if stderr != "" {
						log.Fatalf("Error: %s", stderr)
					} else {
						log.Println(stdout)
					}
				case "ubuntu-2004":
					flags := flagsK8s
					for k, v := range defaultAMIBuildConfig.Ubuntu2004 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}

					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					log.Println(fmt.Sprintf("Info: flags:  \"%s\"", flags))

					stdout, stderr, err := custom.Shell(fmt.Sprintf("cd image-builder/images/capi && PACKER_FLAGS=\"%s\" make build-ami-%s", flags, os))
					custom.CheckError(err, stderr)
					if stderr != "" {
						log.Fatalf("Error: %s", stderr)
					} else {
						log.Println(stdout)
					}
				default:
					log.Println(fmt.Sprintf("Warning: Invalid OS %s. Skipping image building.", os))
				}
			}

			if *cleanup {
				for _, r := range allRegions {
					log.Println("Info: Creating new instance of EC2 client with region", r)
					svc := ec2.New(mySession, aws.NewConfig().WithRegion(r))
					log.Println("Info: New instance of EC2 client with region", r, "created successfully ")

					descImgInput := &ec2.DescribeImagesInput{
						Owners: []*string{&ownerID},
					}

					log.Println("Info: Checking for temporary CAPA AMIs in", r)
					l, _ := svc.DescribeImages(descImgInput)
					ami_snap := map[string][]string{}

					for _, img := range l.Images {
						if strings.HasPrefix(*img.Name, "test-capa-ami-") {
							snapshotList := []string{}
							for _, dev := range img.BlockDeviceMappings {
								ami_snap[*img.ImageId] = append(snapshotList, *dev.Ebs.SnapshotId)
							}
						}
					}

					for ami, snaps := range ami_snap {
						deregImgInput := ec2.DeregisterImageInput{
							ImageId: &ami,
						}

						log.Println("Info: Deregistering AMI:", ami)
						_, err := svc.DeregisterImage(&deregImgInput)
						if err != nil {
							log.Fatal("Error:", err)
						}
						log.Println("Info: AMI", ami, "deregistered successfully")

						for _, snap := range snaps {
							delSnapInput := ec2.DeleteSnapshotInput{
								SnapshotId: &snap,
							}

							log.Println("Info: Deleting snapshot:", snap)
							_, err := svc.DeleteSnapshot(&delSnapInput)
							if err != nil {
								log.Fatal("Error:", err)
							}
							log.Println("Info: Snapshot", snap, "deleted successfully")
						}
					}
				}
			}
		} else {
			log.Printf("Info: AMI for Kubernetes %s already exists. Skipping image building.", v)
		}
	}
}
