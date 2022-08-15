package main

import (
	"context"

	"github.com/aws/karpenter/pkg/cloudprovider"
	"github.com/aws/karpenter/pkg/controllers"
	"github.com/bwagner5/karpenter-k3d/pkg/k3dp"
)

func main() {
	controllers.Initialize(func(ctx context.Context, options cloudprovider.Options) cloudprovider.CloudProvider {
		return k3dp.NewCloudProvider(ctx, options)
	})
}
