package e2e

import (
	"context"
	"path/filepath"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	// 假设你的 CRD API 在 `mydomain/api/v1` 包中
	myapiv1 "github.com/lyzhang1999/application/api/v1"
	v1 "github.com/lyzhang1999/application/api/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// 定义全局变量来存储环境和客户端
var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func(done Done) {
	// 设置日志
	ctrl.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	// 初始化 envtest 环境
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"), // 指向你的 CRD yaml 文件路径
		},
	}

	var err error
	// 启动测试环境
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	// 注册你的自定义资源
	err = myapiv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// 创建 Kubernetes 客户端
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	close(done)
})

var _ = AfterSuite(func() {
	// 停止 envtest 环境
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("CRD Test", func() {
	ctx := context.Background()

	It("should create and fetch a custom resource", func() {
		// 创建一个自定义资源对象
		myResource := &v1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "application-sample",
				Namespace: "default",
			},
			Spec: v1.ApplicationSpec{
				Deployment: v1.ApplicationDeployment{
					Image:    "nginx",
					Replicas: 1,
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
					IngressClassName: func(s string) *string { return &s }("nginx"),
					Rules: []networkingv1.IngressRule{
						{
							Host: "example.foo.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     "/",
											PathType: func(pt networkingv1.PathType) *networkingv1.PathType { return &pt }(networkingv1.PathTypePrefix),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "application-sample",
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

		// 创建该资源
		Expect(k8sClient.Create(ctx, myResource)).Should(Succeed())

		// 获取资源
		fetchedResource := &myapiv1.Application{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      "application-sample",
			Namespace: "default",
		}, fetchedResource)).Should(Succeed())

		// 验证资源的字段
		// Expect(fetchedResource.Spec.Field1).To(Equal("value1"))
		// Expect(fetchedResource.Spec.Field2).To(Equal("value2"))
	})

})
