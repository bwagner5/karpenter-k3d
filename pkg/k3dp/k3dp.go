package k3dp

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/aws/karpenter/pkg/apis/provisioning/v1alpha5"
	"github.com/aws/karpenter/pkg/cloudprovider"
	"github.com/aws/karpenter/pkg/utils/injection"
	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type K3DCloudProvider struct {
	cluster    *types.Cluster
	kubeClient kclient.Client
}

func NewCloudProvider(ctx context.Context, opts cloudprovider.Options) cloudprovider.CloudProvider {
	ctx = logging.WithLogger(ctx, logging.FromContext(ctx).Named("k3d"))
	cluster, err := client.ClusterGet(ctx, runtimes.SelectedRuntime, &types.Cluster{Name: injection.GetOptions(ctx).ClusterName})
	if err != nil {
		logging.FromContext(ctx).Errorf("getting k3d cluster %s, %v", injection.GetOptions(ctx).ClusterName, err)
	}
	return &K3DCloudProvider{
		cluster:    cluster,
		kubeClient: opts.KubeClient,
	}
}

func (c *K3DCloudProvider) Create(ctx context.Context, nodeRequest *cloudprovider.NodeRequest) (*v1.Node, error) {
	it := lo.MinBy(nodeRequest.InstanceTypeOptions, func(it1 cloudprovider.InstanceType, it2 cloudprovider.InstanceType) bool {
		mem1 := it1.Resources()[v1.ResourceMemory]
		mem2 := it2.Resources()[v1.ResourceMemory]
		return mem1.Value() < mem2.Value()
	})
	mem := it.Resources()[v1.ResourceMemory]
	name := fmt.Sprintf("node-%d", rand.Int())
	if err := client.NodeAddToCluster(ctx, runtimes.SelectedRuntime, &types.Node{
		Name:          name,
		Role:          "agent",
		Image:         "rancher/k3s:v1.23.8-k3s1",
		Memory:        mem.String(),
		K3sNodeLabels: map[string]string{v1.LabelInstanceTypeStable: it.Name()},
	}, c.cluster, types.NodeCreateOpts{Wait: true}); err != nil {
		logging.FromContext(ctx).Errorf("creating k3d node %s, %v", name, err)
		return nil, err
	}
	labels := map[string]string{}
	n := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: v1.NodeSpec{
			ProviderID: fmt.Sprintf("k3d://%s", name),
		},
	}
	return n, nil
}

func (c *K3DCloudProvider) Delete(ctx context.Context, n *v1.Node) error {
	if err := client.NodeDelete(ctx, runtimes.SelectedRuntime, &types.Node{Name: n.Name}, types.NodeDeleteOpts{}); err != nil {
		return err
	}
	if err := c.kubeClient.Delete(ctx, n); err != nil {
		return err
	}
	return nil
}

func (c *K3DCloudProvider) GetInstanceTypes(_ context.Context, provisioner *v1alpha5.Provisioner) ([]cloudprovider.InstanceType, error) {
	return []cloudprovider.InstanceType{
		&LocalInstanceType{Options: LocalInstanceTypeOptions{
			Name:  "k3s",
			Price: 0,
			Resources: v1.ResourceList{
				v1.ResourceCPU:              resource.MustParse("1"),
				v1.ResourceMemory:           resource.MustParse("128Mi"),
				v1.ResourceEphemeralStorage: resource.MustParse("256Mi"),
				v1.ResourcePods:             resource.MustParse("3"),
			},
			Overhead: v1.ResourceList{
				v1.ResourceCPU:              resource.MustParse("10m"),
				v1.ResourceMemory:           resource.MustParse("10Mi"),
				v1.ResourceEphemeralStorage: resource.MustParse("128Mi"),
			},
			Architecture:    "arm64",
			OperatingSystem: "linux",
			Offerings: []cloudprovider.Offering{
				{
					CapacityType: "on-demand",
					Zone:         "zone-1",
				},
				{
					CapacityType: "on-demand",
					Zone:         "zone-2",
				},
				{
					CapacityType: "on-demand",
					Zone:         "zone-3",
				},
			},
		}},
	}, nil
}

// Name returns the CloudProvider implementation name.
func (c *K3DCloudProvider) Name() string {
	return "k3d"
}
