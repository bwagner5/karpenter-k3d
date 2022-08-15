package main

import (
	"context"

	"github.com/aws/karpenter/pkg/cloudprovider"
	"github.com/aws/karpenter/pkg/webhooks"
	"github.com/bwagner5/karpenter-k3d/pkg/k3dp"
)

func main() {
	webhooks.Initialize(func(ctx context.Context, options cloudprovider.Options) cloudprovider.CloudProvider {
		return k3dp.NewCloudProvider(ctx, options)
	})
}
