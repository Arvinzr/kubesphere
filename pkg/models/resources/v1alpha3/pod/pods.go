/*

 Copyright 2019 The KubeSphere Authors.

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
package pod

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"kubesphere.io/kubesphere/pkg/api"
	"kubesphere.io/kubesphere/pkg/apiserver/query"
	"kubesphere.io/kubesphere/pkg/models/resources/v1alpha3"
)

type podsGetter struct {
	informer informers.SharedInformerFactory
}

func New(sharedInformers informers.SharedInformerFactory) v1alpha3.Interface {
	return &podsGetter{informer: sharedInformers}
}

func (d *podsGetter) Get(namespace, name string) (runtime.Object, error) {
	return d.informer.Core().V1().Pods().Lister().Pods(namespace).Get(name)
}

func (d *podsGetter) List(namespace string, query *query.Query) (*api.ListResult, error) {

	all, err := d.informer.Core().V1().Pods().Lister().Pods(namespace).List(query.Selector())
	if err != nil {
		return nil, err
	}

	var result []runtime.Object
	for _, app := range all {
		result = append(result, app)
	}

	return v1alpha3.DefaultList(result, query, d.compare, d.filter), nil
}

func (d *podsGetter) compare(left runtime.Object, right runtime.Object, field query.Field) bool {

	leftPod, ok := left.(*corev1.Pod)
	if !ok {
		return false
	}

	rightPod, ok := right.(*corev1.Pod)
	if !ok {
		return false
	}

	return v1alpha3.DefaultObjectMetaCompare(leftPod.ObjectMeta, rightPod.ObjectMeta, field)
}

func (d *podsGetter) filter(object runtime.Object, filter query.Filter) bool {
	pod, ok := object.(*corev1.Pod)

	if !ok {
		return false
	}
	switch filter.Field {
	case "nodeName":
		if pod.Spec.NodeName != string(filter.Value) {
			return false
		}
	case "pvcName":
		if !d.podBindPVC(pod, string(filter.Value)) {
			return false
		}
	case "serviceName":
		if !d.podBelongToService(pod, string(filter.Value)) {
			return false
		}
	}
	return v1alpha3.DefaultObjectMetaFilter(pod.ObjectMeta, filter)
}

func (s *podsGetter) podBindPVC(item *corev1.Pod, pvcName string) bool {
	for _, v := range item.Spec.Volumes {
		if v.VolumeSource.PersistentVolumeClaim != nil &&
			v.VolumeSource.PersistentVolumeClaim.ClaimName == pvcName {
			return true
		}
	}
	return false
}

func (s *podsGetter) podBelongToService(item *corev1.Pod, serviceName string) bool {
	service, err := s.informer.Core().V1().Services().Lister().Services(item.Namespace).Get(serviceName)
	if err != nil {
		return false
	}
	selector := labels.Set(service.Spec.Selector).AsSelectorPreValidated()
	if selector.Empty() || !selector.Matches(labels.Set(item.Labels)) {
		return false
	}
	return true
}
