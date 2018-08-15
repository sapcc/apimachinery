package v1alpha1

import (
	"fmt"
	"reflect"

	"github.com/appscode/go/log"
	crdutils "github.com/appscode/kutil/apiextensions/v1beta1"
	meta_util "github.com/appscode/kutil/meta"
	"github.com/golang/glog"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	mona "kmodules.xyz/monitoring-agent-api/api/v1"
)

func (r Redis) OffshootName() string {
	return r.Name
}

func (r Redis) OffshootSelectors() map[string]string {
	return map[string]string{
		LabelDatabaseName: r.Name,
		LabelDatabaseKind: ResourceKindRedis,
	}
}

func (r Redis) OffshootLabels() map[string]string {
	return filterTags(r.OffshootSelectors(), r.Labels)
}

func (r Redis) ResourceShortCode() string {
	return ResourceCodeRedis
}

func (r Redis) ResourceKind() string {
	return ResourceKindRedis
}

func (r Redis) ResourceSingular() string {
	return ResourceSingularRedis
}

func (r Redis) ResourcePlural() string {
	return ResourcePluralRedis
}

func (r Redis) ServiceName() string {
	return r.OffshootName()
}

type redisStatsService struct {
	*Redis
}

func (r redisStatsService) GetNamespace() string {
	return r.Redis.GetNamespace()
}

func (r redisStatsService) ServiceName() string {
	return r.OffshootName() + "-stats"
}

func (r redisStatsService) ServiceMonitorName() string {
	return fmt.Sprintf("kubedb-%s-%s", r.Namespace, r.Name)
}

func (r redisStatsService) Path() string {
	return fmt.Sprintf("/kubedb.com/v1alpha1/namespaces/%s/%s/%s/metrics", r.Namespace, r.ResourcePlural(), r.Name)
}

func (r redisStatsService) Scheme() string {
	return ""
}

func (r Redis) StatsService() mona.StatsAccessor {
	return &redisStatsService{&r}
}

func (r *Redis) GetMonitoringVendor() string {
	if r.Spec.Monitor != nil {
		return r.Spec.Monitor.Agent.Vendor()
	}
	return ""
}

func (r Redis) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Plural:        ResourcePluralRedis,
		Singular:      ResourceSingularRedis,
		Kind:          ResourceKindRedis,
		ShortNames:    []string{ResourceCodeRedis},
		Categories:    []string{"datastore", "kubedb", "appscode"},
		ResourceScope: string(apiextensions.NamespaceScoped),
		Versions: []apiextensions.CustomResourceDefinitionVersion{
			{
				Name:    SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
		},
		Labels: crdutils.Labels{
			LabelsMap: map[string]string{"app": "kubedb"},
		},
		SpecDefinitionName:      "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1.Redis",
		EnableValidation:        true,
		GetOpenAPIDefinitions:   GetOpenAPIDefinitions,
		EnableStatusSubresource: EnableStatusSubresource,
		AdditionalPrinterColumns: []apiextensions.CustomResourceColumnDefinition{
			{
				Name:     "Version",
				Type:     "string",
				JSONPath: ".spec.version",
			},
			{
				Name:     "Status",
				Type:     "string",
				JSONPath: ".status.phase",
			},
			{
				Name:     "Age",
				Type:     "date",
				JSONPath: ".metadata.creationTimestamp",
			},
		},
	}, setNameSchema)
}

func (r *Redis) Migrate() {
	if r == nil {
		return
	}
	r.Spec.Migrate()
}

func (r *RedisSpec) Migrate() {
	if r == nil {
		return
	}
	if len(r.NodeSelector) > 0 {
		r.PodTemplate.Spec.NodeSelector = r.NodeSelector
		r.NodeSelector = nil
	}
	if r.Resources != nil {
		r.PodTemplate.Spec.Resources = *r.Resources
		r.Resources = nil
	}
	if r.Affinity != nil {
		r.PodTemplate.Spec.Affinity = r.Affinity
		r.Affinity = nil
	}
	if len(r.SchedulerName) > 0 {
		r.PodTemplate.Spec.SchedulerName = r.SchedulerName
		r.SchedulerName = ""
	}
	if len(r.Tolerations) > 0 {
		r.PodTemplate.Spec.Tolerations = r.Tolerations
		r.Tolerations = nil
	}
	if len(r.ImagePullSecrets) > 0 {
		r.PodTemplate.Spec.ImagePullSecrets = r.ImagePullSecrets
		r.ImagePullSecrets = nil
	}
}

func (r *Redis) Equal(other *Redis) bool {
	if EnableStatusSubresource {
		// At this moment, metadata.Generation is incremented only by `spec`.
		// issue tracked: https://github.com/kubernetes/kubernetes/issues/67428
		// So look for changes in metadata.labels as well.
		if r.Generation <= r.Status.ObservedGeneration && reflect.DeepEqual(other.Labels, r.Labels) {
			return true
		}
		if glog.V(log.LevelDebug) {
			diff := meta_util.Diff(other, r)
			glog.InfoDepth(1, "meta.Generation [%d] is higher than status.observedGeneration [%d] in Redis %s/%s with Diff: %s",
				r.Generation, r.Status.ObservedGeneration, r.Namespace, r.Name, diff)
		}
		return false
	}

	if !meta_util.Equal(other.Spec, r.Spec) || !reflect.DeepEqual(other.Labels, r.Labels) {
		if glog.V(log.LevelDebug) {
			diff := meta_util.Diff(other, r)
			glog.InfoDepth(1, "Redis %s/%s has changed. Diff: %s", r.Namespace, r.Name, diff)
		}
		return false
	}
	return true
}
