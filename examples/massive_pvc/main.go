/*
Copyright 2017 The Kubernetes Authors.

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

// Note: the example only works with the code within the same release/branch.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	//
	// Uncomment to load all auth plugins
	//
	// Or uncomment to load specific auth plugins

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

func createPVCs(pvcClient v1.PersistentVolumeClaimInterface, start, end int) {
	volumeMode := apiv1.PersistentVolumeFilesystem
	for i := start; i <= end; i++ {
		pvc := &apiv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pvc-" + strconv.Itoa(i),
			},
			Spec: apiv1.PersistentVolumeClaimSpec{
				AccessModes: []apiv1.PersistentVolumeAccessMode{
					apiv1.ReadWriteOnce,
				},
				Resources: apiv1.ResourceRequirements{
					Requests: apiv1.ResourceList{
						"storage": resource.MustParse("1Gi"),
					},
				},
				VolumeMode: &volumeMode,
			},
		}

		result, err := pvcClient.Create(context.TODO(), pvc, metav1.CreateOptions{})
		if err != nil {
			panic(err)
		}
		fmt.Printf("Created pvc %q.\n", result.GetObjectMeta().GetName())
	}
}

func deletePVCs(pvcClient v1.PersistentVolumeClaimInterface, start, end int) {
	deletePolicy := metav1.DeletePropagationForeground
	for i := start; i <= end; i++ {
		if err := pvcClient.Delete(context.TODO(), "pvc-"+strconv.Itoa(i), metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}); err != nil {
			panic(err)
		}
	}
}

func createPod(podClient v1.PodInterface, start, end int) {
	privileged := true

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hold-massive-pvcs",
		},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{
					Name:            "hold-massive-pvcs",
					Image:           "ubuntu:latest",
					ImagePullPolicy: apiv1.PullIfNotPresent,
					Command: []string{
						"/bin/sleep",
						"3600",
					},
					SecurityContext: &apiv1.SecurityContext{
						Privileged: &privileged,
					},
				},
			},
		},
	}

	for i := start; i <= end; i++ {
		vol := apiv1.Volume{
			Name: "pvc-" + strconv.Itoa(i),
			VolumeSource: apiv1.VolumeSource{
				PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
					ClaimName: "pvc-" + strconv.Itoa(i),
				},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, vol)

		volumemnt := apiv1.VolumeMount{
			Name:      "pvc-" + strconv.Itoa(i),
			MountPath: "/" + "pvc-" + strconv.Itoa(i),
		}
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, volumemnt)
	}

	result, err := podClient.Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created pod %q.\n", result.GetObjectMeta().GetName())
}

func deletePod(podClient v1.PodInterface) {
	deletePolicy := metav1.DeletePropagationForeground
	if err := podClient.Delete(context.TODO(), "hold-massive-pvcs", metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	start := flag.Int("start", 0, "start index")
	end := flag.Int("end", 5, "end index")
	flag.Parse()

	fmt.Printf("start %v end %v\n", *start, *end)

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	pvcClient := clientset.CoreV1().PersistentVolumeClaims(apiv1.NamespaceDefault)
	podClient := clientset.CoreV1().Pods(apiv1.NamespaceDefault)

	// Create pvc
	fmt.Println("Creating pvc...")
	createPVCs(pvcClient, *start, *end)

	// List pvc
	prompt()
	fmt.Printf("Listing pvc in namespace %q:\n", apiv1.NamespaceDefault)
	pvclist, err := pvcClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, pvc := range pvclist.Items {
		fmt.Printf(" pvc %s\n", pvc.Name)
	}

	// List create pod
	prompt()
	fmt.Println("Creating pod...")
	createPod(podClient, *start, *end)

	// Delete pod
	prompt()
	fmt.Println("Deleting pod...")
	deletePod(podClient)
	fmt.Println("Deleted deployment.")

	// Delete pvc
	prompt()
	fmt.Println("Deleting pvc...")
	deletePVCs(pvcClient, *start, *end)
	fmt.Println("Deleted pvc.")
}

func prompt() {
	fmt.Printf("-> Press Return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Println()
}

//func int32Ptr(i int32) *int32 { return &i }
