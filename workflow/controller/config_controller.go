package controller

import (
	"context"
	"fmt"
	"io/ioutil"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/cyrusbiotechnology/argo/errors"
	"github.com/cyrusbiotechnology/argo/workflow/common"
	"github.com/cyrusbiotechnology/argo/workflow/config"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

// ResyncConfig reloads the controller config from the configmap
func (wfc *WorkflowController) ResyncConfig() error {

	if wfc.configFile != "" {
		log.Infof("Loading configfile from %s", wfc.configFile)
		return wfc.updateConfigFromFile(wfc.configFile)
	} else {
		cmClient := wfc.kubeclientset.CoreV1().ConfigMaps(wfc.namespace)
		cm, err := cmClient.Get(wfc.configMap, metav1.GetOptions{})
		if err != nil {
			return errors.InternalWrapError(err)
		}
		return wfc.updateConfig(cm)
	}
}

func (wfc *WorkflowController) updateConfigFromFile(filePath string) error {
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Errorf("Error reading config file %s", filePath)
		return err
	}
	return wfc.updateConfig(string(fileData))

}

func (wfc *WorkflowController) updateConfig(cm *apiv1.ConfigMap) error {
	configStr, ok := cm.Data[common.WorkflowControllerConfigMapKey]
	if !ok {
		log.Warnf("ConfigMap '%s' does not have key '%s'", wfc.configMap, common.WorkflowControllerConfigMapKey)
		return nil
	}
	var config config.WorkflowControllerConfig
	err := yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	log.Printf("workflow controller configuration from %s:\n%s", wfc.configMap, configStr)
	if wfc.cliExecutorImage == "" && config.ExecutorImage == "" {
		return errors.Errorf(errors.CodeBadRequest, "ConfigMap '%s' does not have executorImage", wfc.configMap)
	}
	wfc.Config = config

	if wfc.Config.Persistence != nil {
		log.Info("Persistence configuration enabled")
		wfc.wfDBctx, err = wfc.createPersistenceContext()
		if err != nil {
			log.Errorf("Error Creating Persistence context. %v", err)
		} else {
			log.Info("Persistence Session created successfully")
		}
	} else {
		log.Info("Persistence configuration disabled")
		wfc.wfDBctx = nil
	}
	wfc.throttler.SetParallelism(config.Parallelism)
	return nil
}

// executorImage returns the image to use for the workflow executor
func (wfc *WorkflowController) executorImage() string {
	if wfc.cliExecutorImage != "" {
		return wfc.cliExecutorImage
	}
	return wfc.Config.ExecutorImage
}

// executorImagePullPolicy returns the imagePullPolicy to use for the workflow executor
func (wfc *WorkflowController) executorImagePullPolicy() apiv1.PullPolicy {
	if wfc.cliExecutorImagePullPolicy != "" {
		return apiv1.PullPolicy(wfc.cliExecutorImagePullPolicy)
	} else if wfc.Config.Executor != nil && wfc.Config.Executor.ImagePullPolicy != "" {
		return wfc.Config.Executor.ImagePullPolicy
	} else {
		return apiv1.PullPolicy(wfc.Config.ExecutorImagePullPolicy)
	}
}

func (wfc *WorkflowController) watchControllerConfigMap(ctx context.Context) (cache.Controller, error) {
	source := wfc.newControllerConfigMapWatch()
	_, controller := cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*apiv1.ConfigMap); ok {
					log.Infof("Detected ConfigMap update. Updating the controller config.")
					err := wfc.updateConfig(cm)
					if err != nil {
						log.Errorf("Update of config failed due to: %v", err)
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldCM := old.(*apiv1.ConfigMap)
				newCM := new.(*apiv1.ConfigMap)
				if oldCM.ResourceVersion == newCM.ResourceVersion {
					return
				}
				if newCm, ok := new.(*apiv1.ConfigMap); ok {
					log.Infof("Detected ConfigMap update. Updating the controller config.")
					err := wfc.updateConfig(newCm)
					if err != nil {
						log.Errorf("Update of config failed due to: %v", err)
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (wfc *WorkflowController) newControllerConfigMapWatch() *cache.ListWatch {
	c := wfc.kubeclientset.CoreV1().RESTClient()
	resource := "configmaps"
	name := wfc.configMap
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(wfc.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := c.Get().
			Namespace(wfc.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func ReadConfigMapValue(clientset kubernetes.Interface, namespace string, name string, key string) (string, error) {
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	value, ok := cm.Data[key]
	if !ok {
		return "", errors.InternalErrorf("Key %s was not found in the %s configMap.", key, name)
	}
	return value, nil
}

func getArtifactRepositoryRef(wfc *WorkflowController, configMapName string, key string) (*config.ArtifactRepository, error) {
	configStr, err := ReadConfigMapValue(wfc.kubeclientset, wfc.namespace, configMapName, key)
	if err != nil {
		return nil, err
	}
	var config config.ArtifactRepository
	err = yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &config, nil
}
