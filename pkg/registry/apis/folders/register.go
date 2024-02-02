package folders

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	common "k8s.io/kube-openapi/pkg/common"

	"github.com/grafana/grafana/pkg/apis/folder/v0alpha1"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	"github.com/grafana/grafana/pkg/services/apiserver/endpoints/request"
	grafanarest "github.com/grafana/grafana/pkg/services/apiserver/rest"
	"github.com/grafana/grafana/pkg/services/apiserver/utils"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/folder"
	"github.com/grafana/grafana/pkg/setting"
)

var _ builder.APIGroupBuilder = (*FolderAPIBuilder)(nil)

var resourceInfo = v0alpha1.FolderResourceInfo

// This is used just so wire has something unique to return
type FolderAPIBuilder struct {
	gv         schema.GroupVersion
	features   *featuremgmt.FeatureManager
	namespacer request.NamespaceMapper
	folderSvc  folder.Service
}

func RegisterAPIService(cfg *setting.Cfg,
	features *featuremgmt.FeatureManager,
	apiregistration builder.APIRegistrar,
	folderSvc folder.Service,
) *FolderAPIBuilder {
	if !features.IsEnabledGlobally(featuremgmt.FlagGrafanaAPIServerWithExperimentalAPIs) {
		return nil // skip registration unless opting into experimental apis
	}

	builder := &FolderAPIBuilder{
		gv:         resourceInfo.GroupVersion(),
		features:   features,
		namespacer: request.GetNamespaceMapper(cfg),
		folderSvc:  folderSvc,
	}
	apiregistration.RegisterAPI(builder)
	return builder
}

func (b *FolderAPIBuilder) GetGroupVersion() schema.GroupVersion {
	return b.gv
}

func addKnownTypes(scheme *runtime.Scheme, gv schema.GroupVersion) {
	scheme.AddKnownTypes(gv,
		&v0alpha1.Folder{},
		&v0alpha1.FolderList{},
		&v0alpha1.FolderInfoList{},
	)
}

func (b *FolderAPIBuilder) InstallSchema(scheme *runtime.Scheme) error {
	addKnownTypes(scheme, b.gv)

	// Link this version to the internal representation.
	// This is used for server-side-apply (PATCH), and avoids the error:
	//   "no kind is registered for the type"
	addKnownTypes(scheme, schema.GroupVersion{
		Group:   b.gv.Group,
		Version: runtime.APIVersionInternal,
	})

	// If multiple versions exist, then register conversions from zz_generated.conversion.go
	// if err := playlist.RegisterConversions(scheme); err != nil {
	//   return err
	// }
	metav1.AddToGroupVersion(scheme, b.gv)
	return scheme.SetVersionPriority(b.gv)
}

func (b *FolderAPIBuilder) GetAPIGroupInfo(
	scheme *runtime.Scheme,
	codecs serializer.CodecFactory, // pointer?
	optsGetter generic.RESTOptionsGetter,
	dualWrite bool,
) (*genericapiserver.APIGroupInfo, error) {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(v0alpha1.GROUP, scheme, metav1.ParameterCodec, codecs)

	legacyStore := &legacyStorage{
		service:    b.folderSvc,
		namespacer: b.namespacer,
		tableConverter: utils.NewTableConverter(
			resourceInfo.GroupResource(),
			[]metav1.TableColumnDefinition{
				{Name: "Name", Type: "string", Format: "name"},
				{Name: "Title", Type: "string", Format: "string", Description: "The display name"},
				{Name: "Parent", Type: "string", Format: "string", Description: "Parent folder UID"},
			},
			func(obj any) ([]interface{}, error) {
				r, ok := obj.(*v0alpha1.Folder)
				if ok {
					accessor, _ := utils.MetaAccessor(r)
					return []interface{}{
						r.Name,
						r.Spec.Title,
						accessor.GetFolder(),
					}, nil
				}
				return nil, fmt.Errorf("expected resource or info")
			}),
	}

	storage := map[string]rest.Storage{}
	storage[resourceInfo.StoragePath()] = legacyStore
	storage[resourceInfo.StoragePath("parents")] = &subParentsREST{b.folderSvc}
	storage[resourceInfo.StoragePath("children")] = &subChildrenREST{b.folderSvc}

	// enable dual writes if a RESTOptionsGetter is provided
	if dualWrite && optsGetter != nil {
		store, err := newStorage(scheme, optsGetter, legacyStore)
		if err != nil {
			return nil, err
		}
		storage[resourceInfo.StoragePath()] = grafanarest.NewDualWriter(legacyStore, store)
	}

	apiGroupInfo.VersionedResourcesStorageMap[v0alpha1.VERSION] = storage
	return &apiGroupInfo, nil
}

func (b *FolderAPIBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions {
	return v0alpha1.GetOpenAPIDefinitions
}

func (b *FolderAPIBuilder) GetAPIRoutes() *builder.APIRoutes {
	return nil // no custom API routes
}

func (b *FolderAPIBuilder) GetAuthorizer() authorizer.Authorizer {
	return nil // TODO: the FGAC rules encoded in the service can be moved here
}
