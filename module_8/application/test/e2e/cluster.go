package e2e

import (
	"context"
	"path/filepath"
	"testing"

	myapiv1 "k8sbuilder-demo/api/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func(done Done) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "cofig", "crd", "bases"),
		},
	}

	var err error
	// 启动 envtest环境
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	// 注册 自定义资源
	err = myapiv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// 创建 K8s 客户端
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())
	close(done)
})

var _ = AfterSuite(func() {
	// 关闭 envtest 环境
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("CRD Test", func() {
	ctx := context.Background()
	It("should create and fetch a CRD", func() {
		//  构造自定义资源
		myRousrouce := &myapiv1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-application",
				Namespace: "default",
			},
			Spec: myapiv1.ApplicationSpec{
				Deployment: myapiv1.ApplicationDeployment{
					Replicas: 1,
					Image:    "nginx",
					Port:     80,
				},
				Service: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
						},
					},
				},
				Ingress: networkingv1.IngressSpec{
					IngressClassName: "nginx",
					Rules: []networkingv1.IngressRule{
						{
							Host: "example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/",
											PathType: networkingv1.PathTypePrefix,
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "test-application",
													Port: networkingv1.ServiceBackendPort{
														Number: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		//  创建资源
		Expect(k8sClient.Create(ctx, myRousrouce)).Should(Succeed())
		// 获取资源
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      "test-application",
			Namespace: "default",
		}, fetchedResource)).Should(Succeed())
	})
})
