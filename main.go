package main

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/jessevdk/go-flags"
	cache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/webdevops/azure-resourcegraph-exporter/config"
	"github.com/webdevops/azure-resourcegraph-exporter/kusto"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

const (
	Author = "webdevops.io"
)

var (
	argparser *flags.Parser
	opts      config.Opts

	Config kusto.Config

	AzureAuthorizer    autorest.Authorizer
	AzureSubscriptions []subscriptions.Subscription
	AzureEnvironment   azure.Environment

	metricCache *cache.Cache

	// Git version information
	gitCommit = "<unknown>"
	gitTag    = "<unknown>"
)

func main() {
	initArgparser()

	log.Infof("starting azure-resourcegraph-exporter v%s (%s; %s; by %v)", gitTag, gitCommit, runtime.Version(), Author)
	log.Info(string(opts.GetJson()))
	initGlobalMetrics()

	metricCache = cache.New(120*time.Second, 60*time.Second)

	log.Infof("loading config")
	readConfig()

	log.Infof("init Azure")
	initAzureConnection()

	log.Infof("starting http server on %s", opts.ServerBind)
	startHttpServer()
}

// init argparser and parse/validate arguments
func initArgparser() {
	argparser = flags.NewParser(&opts, flags.Default)
	_, err := argparser.Parse()

	// check if there is an parse error
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			fmt.Println()
			argparser.WriteHelp(os.Stdout)
			os.Exit(1)
		}
	}

	// verbose level
	if opts.Logger.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	// debug level
	if opts.Logger.Debug {
		log.SetReportCaller(true)
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				s := strings.Split(f.Function, ".")
				funcName := s[len(s)-1]
				return funcName, fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
			},
		})
	}

	// json log format
	if opts.Logger.LogJson {
		log.SetReportCaller(true)
		log.SetFormatter(&log.JSONFormatter{
			DisableTimestamp: true,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				s := strings.Split(f.Function, ".")
				funcName := s[len(s)-1]
				return funcName, fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
			},
		})
	}
}

func readConfig() {
	Config = kusto.NewConfig(opts.Config.Path)

	if err := Config.Validate(); err != nil {
		log.Panic(err)
	}
}

// Init and build Azure authorzier
func initAzureConnection() {
	var err error
	ctx := context.Background()

	// setup azure authorizer
	AzureAuthorizer, err = auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Panic(err)
	}
	subscriptionsClient := subscriptions.NewClient()
	subscriptionsClient.Authorizer = AzureAuthorizer

	if len(opts.Azure.Subscription) == 0 {
		// auto lookup subscriptions
		listResult, err := subscriptionsClient.List(ctx)
		if err != nil {
			log.Panic(err)
		}
		AzureSubscriptions = listResult.Values()

		if len(AzureSubscriptions) == 0 {
			log.Panic("no Azure Subscriptions found via auto detection, does this ServicePrincipal have read permissions to the subscriptions?")
		}
	} else {
		// fixed subscription list
		AzureSubscriptions = []subscriptions.Subscription{}
		for _, subId := range opts.Azure.Subscription {
			result, err := subscriptionsClient.Get(ctx, subId)
			if err != nil {
				log.Panic(err)
			}
			AzureSubscriptions = append(AzureSubscriptions, result)
		}
	}

	AzureEnvironment, err = azure.EnvironmentFromName(*opts.Azure.Environment)
	if err != nil {
		log.Panic(err)
	}
}

// start and handle prometheus handler
func startHttpServer() {
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/probe", handleProbeRequest)

	log.Fatal(http.ListenAndServe(opts.ServerBind, nil))
}
