package k3dp

import (
	"github.com/aws/karpenter/pkg/apis/provisioning/v1alpha5"
	"github.com/aws/karpenter/pkg/cloudprovider"
	"github.com/aws/karpenter/pkg/cloudprovider/aws/apis/v1alpha1"
	"github.com/aws/karpenter/pkg/scheduling"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
)

type LocalInstanceTypeOptions struct {
	Name            string
	Price           float64
	Resources       v1.ResourceList
	Overhead        v1.ResourceList
	Offerings       []cloudprovider.Offering
	Architecture    string
	OperatingSystem string
}

type LocalInstanceType struct {
	Options LocalInstanceTypeOptions
}

func (i *LocalInstanceType) Name() string {
	return i.Options.Name
}

func (i *LocalInstanceType) Price() float64 {
	if i.Options.Price != 0 {
		return i.Options.Price
	}

	price := 0.0
	for k, v := range i.Resources() {
		switch k {
		case v1.ResourceCPU:
			price += 0.1 * v.AsApproximateFloat64()
		case v1.ResourceMemory:
			price += 0.1 * v.AsApproximateFloat64() / (1e9)
		case v1alpha1.ResourceNVIDIAGPU, v1alpha1.ResourceAMDGPU:
			price += 1.0
		}
	}
	return price
}

func (i *LocalInstanceType) Resources() v1.ResourceList {
	return i.Options.Resources
}

func (i *LocalInstanceType) Offerings() []cloudprovider.Offering {
	return i.Options.Offerings
}

func (i *LocalInstanceType) Overhead() v1.ResourceList {
	return i.Options.Overhead
}

func (i *LocalInstanceType) Requirements() scheduling.Requirements {
	requirements := scheduling.NewRequirements(
		scheduling.NewRequirement(v1.LabelInstanceTypeStable, v1.NodeSelectorOpIn, i.Options.Name),
		scheduling.NewRequirement(v1.LabelArchStable, v1.NodeSelectorOpIn, i.Options.Architecture),
		scheduling.NewRequirement(v1.LabelOSStable, v1.NodeSelectorOpIn, i.Options.OperatingSystem),
		scheduling.NewRequirement(v1.LabelTopologyZone, v1.NodeSelectorOpIn, lo.Map(i.Offerings(), func(o cloudprovider.Offering, _ int) string { return o.Zone })...),
		scheduling.NewRequirement(v1alpha5.LabelCapacityType, v1.NodeSelectorOpIn, lo.Map(i.Offerings(), func(o cloudprovider.Offering, _ int) string { return o.CapacityType })...),
	)
	return requirements
}
