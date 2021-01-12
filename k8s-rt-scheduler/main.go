package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"strconv"
	"github.com/comail/colog"
	"github.com/julienschmidt/httprouter"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/types"
	listersv1 "k8s.io/client-go/listers/core/v1"
	schedulerapi "k8s.io/kubernetes/pkg/scheduler/apis/extender/v1"
)

const (
	versionPath      = "/version"
	apiPrefix        = "/scheduler"
	bindPath         = apiPrefix + "/bind"
	preemptionPath   = apiPrefix + "/preemption"
	predicatesPrefix = apiPrefix + "/predicates"
	prioritiesPrefix = apiPrefix + "/priorities"
)

const (
	ConfigFilePath = "/rt-manager/config.json"
)

const schedulerName = "rt-manager-scheduler"

type predicateFunc func(node *v1.Node, pod *v1.Pod) bool
type priorityFunc func(node *v1.Node, pod *v1.Pod) int

type RTScheduler struct {
	clientset  *kubernetes.Clientset
	//podPriorityQueue pq.PriorityQueue
	nodeLister listersv1.NodeLister
	predicates []predicateFunc
	priorities []priorityFunc
}

var (
	version string

	RTPredicate = Predicate{
		Name: "rt_predicate",
		Func: func(pod v1.Pod, node v1.Node) (bool, error) {
			
			rtquota, err := strconv.ParseFloat(pod.ObjectMeta.Annotations["rt-quota"], 64)
			if err != nil {
				log.Fatal(err)
			}
			
			rtperiod, err := strconv.ParseFloat(pod.ObjectMeta.Annotations["rt-period"], 64)
			if err != nil {
				log.Fatal(err)
			}
			
			
			var utilization = rtquota / rtperiod
			var nodeUtilization = getNodeUtilization(node);
			log.Print(pod.ObjectMeta.Annotations);
			log.Print("Q,P, utilization, NodeUtilization", rtquota, rtperiod, utilization, nodeUtilization);
			
			
			
			return false, nil
		},
	}

	RTPriority = Prioritize{
		Name: "rt_score",
		Func: func(_ v1.Pod, nodes []v1.Node) (*schedulerapi.HostPriorityList, error) {
			var priorityList schedulerapi.HostPriorityList
			priorityList = make([]schedulerapi.HostPriority, len(nodes))
			for i, node := range nodes {
				priorityList[i] = schedulerapi.HostPriority{
					Host:  node.Name,
					Score: 0,
				}
			}
			return &priorityList, nil
		},
	}
	
	NoBind = Bind{
		Func: func(podName string, podNamespace string, podUID types.UID, node string) error {
			return fmt.Errorf("This extender doesn't support Bind.  Please make 'BindVerb' be empty in your ExtenderConfig.")
		},
	}


)

func StringToLevel(levelStr string) colog.Level {
	switch level := strings.ToUpper(levelStr); level {
	case "TRACE":
		return colog.LTrace
	case "DEBUG":
		return colog.LDebug
	case "INFO":
		return colog.LInfo
	case "WARNING":
		return colog.LWarning
	case "ERROR":
		return colog.LError
	case "ALERT":
		return colog.LAlert
	default:
		return colog.LInfo
	}
}


func NewScheduler() RTScheduler {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	return RTScheduler{
		clientset:  clientset,
		//nodeLister: initInformers(clientset, podPriorityQueue, quit),
		predicates: []predicateFunc{
			//enoughUtilizationPredicate,
		},
		priorities: []priorityFunc{
			//randomPriority,
		},
	}
}

func enoughUtilizationPredicate(node *v1.Node, pod *v1.Pod) bool {
	return (pod.ObjectMeta.Name == "kube-gpu-1")
}

func getNodeUtilization(node v1.Node) float64 {
	ann := node.ObjectMeta.Annotations
	res := 0.0
	if ann == nil {
		return 0;
	}
	deployedRT := ann["deployedRT"]
	log.Print(deployedRT);
	
	var containers = strings.Split(deployedRT, ";")
	for cnt, cont := range containers {
		log.Print("container", cont, cnt)
		tmp := strings.ReplaceAll(cont, "(", "")
		tmp = strings.ReplaceAll(tmp, ")", "")
		param := strings.Split(tmp, ",")
		
		period, err := strconv.ParseFloat(param[0], 64)
		if err != nil {
			log.Fatal(err)
		}
		quota, err := strconv.ParseFloat(param[1], 64)
		if err != nil {
			log.Fatal(err)
		}
		utilization := quota/period;
		log.Print(param, utilization)
		res += utilization
	}
	return res
}

func updateNodeUtilization(node *v1.Node, utilization float64) {
	ann := node.ObjectMeta.Annotations
	ann["utilization"] = fmt.Sprintf("%f", utilization);
}

func getPodUtilization(pod *v1.Pod) float64{
	ann := pod.ObjectMeta.Annotations
	if ann == nil {
		return 0;
	}
	return 0;
}




func main() {
	colog.SetDefaultLevel(colog.LInfo)
	colog.SetMinLevel(colog.LInfo)
	colog.SetFormatter(&colog.StdFormatter{
		Colors: true,
		Flag:   log.Ldate | log.Ltime | log.Lshortfile,
	})
	colog.Register()
	level := StringToLevel(os.Getenv("LOG_LEVEL"))
		
	colog.SetMinLevel(level)

	router := httprouter.New()
	AddVersion(router)

	predicates := []Predicate{RTPredicate}
	for _, p := range predicates {
		AddPredicate(router, p)
	}

	priorities := []Prioritize{RTPriority}
	for _, p := range priorities {
		AddPrioritize(router, p)
	}

	AddBind(router, NoBind)

	log.Print("info: server starting on the port! :80")
	if err := http.ListenAndServe(":80", router); err != nil {
		log.Fatal(err)
	}
}
