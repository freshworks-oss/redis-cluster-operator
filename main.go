/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	goruntime "runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	"k8s.io/apimachinery/pkg/runtime"
	runtimeschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	rediskunv1alpha1 "github.com/freshworks/redis-cluster-operator/api/v1alpha1"
	"github.com/freshworks/redis-cluster-operator/controllers/distributedrediscluster"
	clustermanger "github.com/freshworks/redis-cluster-operator/controllers/manager"
	"github.com/freshworks/redis-cluster-operator/controllers/redisclusterbackup"
	"github.com/freshworks/redis-cluster-operator/metrics"
	config2 "github.com/freshworks/redis-cluster-operator/redisconfig"
	utils "github.com/freshworks/redis-cluster-operator/utils/commonutils"
	"github.com/freshworks/redis-cluster-operator/utils/exec"
	"github.com/freshworks/redis-cluster-operator/utils/k8sutil"
	"github.com/freshworks/redis-cluster-operator/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	//+kubebuilder:scaffold:imports
)

var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

var (
	scheme     = runtime.NewScheme()
	setupLog   = ctrl.Log.WithName("setup")
	clusterLog = ctrl.Log.WithName("controller_distributedrediscluster")
)
var log = logf.Log.WithName("main")

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(rediskunv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}
func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", goruntime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", goruntime.GOOS, goruntime.GOARCH))
	//log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
	log.Info(fmt.Sprintf("Version of operator: %s+%s", version.Version, version.GitSHA))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	pflag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	pflag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	pflag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	pflag.CommandLine.AddFlagSet(distributedrediscluster.FlagSet())
	pflag.CommandLine.AddFlagSet(redisclusterbackup.FlagSet())

	// Create a Prometheus registry
	reg := prometheus.DefaultRegisterer

	// Use the NewRecorder function from the metrics package to create an instance of Recorder
	recorder := metrics.NewRecorder("redisoperator", reg)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", 9297), nil); err != nil {
			panic(err)
		}
	}()

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	config2.RedisConf().AddFlags(pflag.CommandLine)
	pflag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	printVersion()
	utils.SetOperatorScope()
	//Obtain leader election ID from env
	leaderElectionID, found := os.LookupEnv("LEADER_ELECTION_ID")
	if !found {
		setupLog.Info("Leader election ID not found in env")
		panic("Pass LEADER_ELECTION_ID as env in operator deployment")
	}
	mgroptions := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	}

	if utils.IsClusterScoped() {
		setupLog.Info("Operator is running in cluster scope")
		err := utils.GenerateNamespaceList(setupLog)
		//This will get the namespace watch and unwatch list but since it is cluster scoped,
		// we need only unwatch list and rest all will be watched.
		if err != nil {
			os.Exit(1)
		}
	} else {
		setupLog.Info("Operator is running in namespace scope")
		err := utils.GenerateNamespaceList(setupLog)
		if err != nil {
			os.Exit(1)
		}

		defaultNamespaces := make(map[string]cache.Config)
		nslist, _ := utils.GetNamespaceList()
		//this will return watch and unwatch list but since it is running in namespace scoped, we need only watch list, so dropping unwatch list.

		for _, ns := range nslist {
			defaultNamespaces[ns] = cache.Config{}
		}
		mgroptions.NewCache = func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			opts.DefaultNamespaces = defaultNamespaces
			return cache.New(config, opts)
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgroptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	distributedredisclusterclient := mgr.GetClient()
	gvk := runtimeschema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}
	httpClient, err := rest.HTTPClientFor(mgr.GetConfig())
	if err != nil {
		os.Exit(1)
	}
	restClient, err := apiutil.RESTClientForGVK(gvk, false, false, mgr.GetConfig(), serializer.NewCodecFactory(clientgoscheme.Scheme), httpClient)
	if err != nil {
		os.Exit(1)
	}
	if err = (&distributedrediscluster.DistributedRedisClusterReconciler{
		Client:                distributedredisclusterclient,
		Scheme:                mgr.GetScheme(),
		Ensurer:               clustermanger.NewEnsureResource(distributedredisclusterclient, clusterLog, recorder),
		StatefulSetController: k8sutil.NewStatefulSetController(distributedredisclusterclient, recorder),
		ServiceController:     k8sutil.NewServiceController(distributedredisclusterclient, recorder),
		PdbController:         k8sutil.NewPodDisruptionBudgetController(distributedredisclusterclient, recorder),
		PvcController:         k8sutil.NewPvcController(distributedredisclusterclient, recorder),
		CrController:          k8sutil.NewCRControl(distributedredisclusterclient, recorder),
		Checker:               clustermanger.NewCheck(distributedredisclusterclient, clusterLog, recorder),
		Execer:                exec.NewRemoteExec(restClient, mgr.GetConfig(), clusterLog),
		MetricsClient:         recorder,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DistributedRedisCluster")
		os.Exit(1)
	}

	redisclusterbackupclient := mgr.GetClient()
	redisclusterbackupdrclient := redisclusterbackup.NewDirectClient(mgr.GetConfig())

	if err = (&redisclusterbackup.RedisClusterBackupReconciler{
		Client:                redisclusterbackupclient,
		Scheme:                mgr.GetScheme(),
		CrController:          k8sutil.NewCRControl(redisclusterbackupclient, recorder),
		StatefulSetController: k8sutil.NewStatefulSetController(redisclusterbackupclient, recorder),
		DirectClient:          redisclusterbackupdrclient,
		JobController:         k8sutil.NewJobController(redisclusterbackupdrclient, recorder),
		DeploymentController:  k8sutil.NewDeploymentController(redisclusterbackupdrclient, recorder),
		ConfigmapController:   k8sutil.NewConfigMapController(redisclusterbackupdrclient, recorder),
		SecretController:      k8sutil.NewSecretController(redisclusterbackupdrclient, recorder),
		Recorder:              mgr.GetEventRecorderFor("redis-cluster-operator-backup"),
		MetricsClient:         recorder,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RedisClusterBackup")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") == "true" {
		log.Info("Starting the WebHook.")
		/*
			ws := mgr.GetWebhookServer()
			ws.CertDir = "/etc/webhook/certs"
			ws.Port = 7443
		*/
		if err = (&rediskunv1alpha1.DistributedRedisCluster{}).SetupWebhookWithManager(mgr); err != nil {
			log.Error(err, "unable to create webHook", "webHook", "DistributedRedisCluster")
			os.Exit(1)
		}
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
