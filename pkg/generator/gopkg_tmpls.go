package generator

const gopkgLockTmpl = `[[projects]]
name = "k8s.io/api"
packages = ["admissionregistration/v1alpha1","apps/v1beta1","apps/v1beta2","authentication/v1","authentication/v1beta1","authorization/v1","authorization/v1beta1","autoscaling/v1","autoscaling/v2beta1","batch/v1","batch/v1beta1","batch/v2alpha1","certificates/v1beta1","core/v1","extensions/v1beta1","networking/v1","policy/v1beta1","rbac/v1","rbac/v1alpha1","rbac/v1beta1","scheduling/v1alpha1","settings/v1alpha1","storage/v1","storage/v1beta1"]
revision = "4df58c811fe2e65feb879227b2b245e4dc26e7ad"
version = "kubernetes-1.8.2"

[[projects]]
name = "k8s.io/apimachinery"
packages = ["pkg/api/equality","pkg/api/errors","pkg/api/meta","pkg/api/resource","pkg/apis/meta/internalversion","pkg/apis/meta/v1","pkg/apis/meta/v1/unstructured","pkg/apis/meta/v1alpha1","pkg/conversion","pkg/conversion/queryparams","pkg/conversion/unstructured","pkg/fields","pkg/labels","pkg/runtime","pkg/runtime/schema","pkg/runtime/serializer","pkg/runtime/serializer/json","pkg/runtime/serializer/protobuf","pkg/runtime/serializer/recognizer","pkg/runtime/serializer/streaming","pkg/runtime/serializer/versioning","pkg/selection","pkg/types","pkg/util/cache","pkg/util/clock","pkg/util/diff","pkg/util/errors","pkg/util/framer","pkg/util/intstr","pkg/util/json","pkg/util/net","pkg/util/runtime","pkg/util/sets","pkg/util/validation","pkg/util/validation/field","pkg/util/wait","pkg/util/yaml","pkg/version","pkg/watch","third_party/forked/golang/reflect"]
revision = "019ae5ada31de202164b118aee88ee2d14075c31"
version = "kubernetes-1.8.2"

[[projects]]
name = "k8s.io/client-go"
packages = ["discovery","discovery/cached","dynamic","kubernetes","kubernetes/scheme","kubernetes/typed/admissionregistration/v1alpha1","kubernetes/typed/apps/v1beta1","kubernetes/typed/apps/v1beta2","kubernetes/typed/authentication/v1","kubernetes/typed/authentication/v1beta1","kubernetes/typed/authorization/v1","kubernetes/typed/authorization/v1beta1","kubernetes/typed/autoscaling/v1","kubernetes/typed/autoscaling/v2beta1","kubernetes/typed/batch/v1","kubernetes/typed/batch/v1beta1","kubernetes/typed/batch/v2alpha1","kubernetes/typed/certificates/v1beta1","kubernetes/typed/core/v1","kubernetes/typed/extensions/v1beta1","kubernetes/typed/networking/v1","kubernetes/typed/policy/v1beta1","kubernetes/typed/rbac/v1","kubernetes/typed/rbac/v1alpha1","kubernetes/typed/rbac/v1beta1","kubernetes/typed/scheduling/v1alpha1","kubernetes/typed/settings/v1alpha1","kubernetes/typed/storage/v1","kubernetes/typed/storage/v1beta1","pkg/version","rest","rest/watch","tools/cache","tools/clientcmd/api","tools/metrics","tools/pager","tools/reference","transport","util/cert","util/flowcontrol","util/integer","util/workqueue"]
revision = "35ccd4336052e7d73018b1382413534936f34eee"
version = "kubernetes-1.8.2"
`

const gopkgTomlTmpl = `[[constraint]]
  name = "k8s.io/api"
  version = "kubernetes-1.8.2"

[[constraint]]
  name = "k8s.io/apimachinery"
  version = "kubernetes-1.8.2"

[[constraint]]
  name = "k8s.io/client-go"
  version = "kubernetes-1.8.2"
`
