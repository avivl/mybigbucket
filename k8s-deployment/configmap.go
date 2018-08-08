// Copyright © 2018 Aviv Laufer <aviv.laufer@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package deployment

import (
	"sync"
	client_v1 "github.com/doitintl/kuberbs/pkg/clientset/v1"
	"github.com/Sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"github.com/google/uuid"
)

const (
	ConfigMapName      = "kuberbs"
	ConfigMapNameSpace = "kube-system"
)

var instance *Configmap
var once sync.Once

func GetInstance(client kubernetes.Interface) *Configmap {
	once.Do(func() {
		instance = newConfigmap(client)
	})
	return instance
}

type Configmap struct {
	lock sync.Mutex
	cfg  *api_v1.ConfigMap
	client kubernetes.Interface
}

func newConfigmap(client kubernetes.Interface) *Configmap {
	configMap, err := client.CoreV1().ConfigMaps(ConfigMapNameSpace).Get(ConfigMapName, meta_v1.GetOptions{})
	if err != nil {
		if err.Error() == "configmaps \"kuberbs\" not found" {
			configMap = &api_v1.ConfigMap{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      ConfigMapName,
					Namespace: ConfigMapNameSpace,
				},
				Data:make(map[string]string),
			}
			configMap.Data["uuid"] = uuid.New().String()
			populate(client ,configMap)
			configMap, err = client.CoreV1().ConfigMaps(ConfigMapNameSpace).Create(configMap)
			if err != nil {
				logrus.Error(err)
				return nil
			}


		}
	}
	return &Configmap{cfg: configMap, client: client}
}

func (c *Configmap) Save() error{
	c.lock.Lock()
	defer c.lock.Unlock()
	var err error
	c.cfg,err =c.client.CoreV1().ConfigMaps(ConfigMapNameSpace).Update(c.cfg)
	if err != nil {
		logrus.Error(err)
	}
	return err
}

func (c *Configmap) Load() *api_v1.ConfigMap {
	c.lock.Lock()
	defer c.lock.Unlock()
	var err error
	c.cfg, err = c.client.CoreV1().ConfigMaps(ConfigMapNameSpace).Get(ConfigMapName, meta_v1.GetOptions{})
	if err != nil {
		logrus.Error(err)
		return nil
	}
	return c.cfg
}


func (c *Configmap) UpdateKeys(data map[string]string) {
	for k, v := range data {
		c.UpdateKey(k,v)
	}
}

func (c *Configmap) UpdateKey(key string, value string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cfg.Data[key] = value
}


func (c *Configmap) GetKey (key string) string {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.cfg.Data[key]
}

func (c *Configmap) DeleteKey (key string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete (c.cfg.Data, key)
}

func (c *Configmap) DeleteKeys(data map[string]string) {
	for k,_  := range data {
		c.DeleteKey(k)
	}
}

func  populate(client kubernetes.Interface,cm *api_v1.ConfigMap) {
	var config *rest.Config
	var err error
	config, err = rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	clientSet, err := client_v1.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	rbs, err := clientSet.Rbs("default").List(meta_v1.ListOptions{})
	if err != nil {
		panic(err)
		return
	}
	for i,ns:= range rbs.Items[0].Spec.Namespaces {
		for _,dp:=range rbs.Items[0].Spec.Namespaces[i].Deployments {
			dd :=NewDeploymentController(client, ns.Name,dp)
			data := dd.Get()
			for k,v:=range  data {
				cm.Data[k]=v
			}

		}
	}
}