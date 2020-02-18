//
// Provides builders and helper methods for preparing Pods and nested Containers
//
package framework

import (
	"bytes"
	"github.com/pkg/errors"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

//
// PodBuilder
//
type PodBuilder struct {
	pod *v1.Pod
}

// NewPodBuilder Creates an instance of a PodBuilder helper
func NewPodBuilder(name string, namespace string) *PodBuilder {
	pb := new(PodBuilder)
	pb.pod = new(v1.Pod)
	pb.pod.Name = name
	pb.pod.Namespace = namespace
	pb.pod.Spec = v1.PodSpec{}
	pb.pod.Status = v1.PodStatus{}
	return pb
}

// NewContainerBuilder creates an instance of a ContainerBuilder helper
func NewContainerBuilder(name string, image string) *ContainerBuilder {
	cb := new(ContainerBuilder)
	cb.c = v1.Container{}
	cb.c.Name = name
	cb.c.Image = image
	cb.c.TerminationMessagePolicy = v1.TerminationMessageFallbackToLogsOnError
	return cb
}

// AddLabel Adds or replaces the given label key and value to Pod
func (p *PodBuilder) AddLabel(key, value string) *PodBuilder {
	if p.pod.Labels == nil {
		p.pod.Labels = map[string]string{}
	}
	p.pod.Labels[key] = value
	return p
}

// AddContainer adds a container to the Pod being prepared
func (p *PodBuilder) AddContainer(c v1.Container) *PodBuilder {
	if p.pod.Spec.Containers == nil {
		p.pod.Spec.Containers = []v1.Container{}
	}
	p.pod.Spec.Containers = append(p.pod.Spec.Containers, c)
	return p
}

// AddConfigMapVolumeSource append a Volume with a local reference
// to a ConfigMap into the Pod Spec
func (p *PodBuilder) AddConfigMapVolumeSource(name string, configMapName string) *PodBuilder {
	if p.pod.Spec.Volumes == nil {
		p.pod.Spec.Volumes = []v1.Volume{}
	}
	v := v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}
	p.pod.Spec.Volumes = append(p.pod.Spec.Volumes, v)
	return p
}

// RestartPolicy defines the RestartPolicy of the Pod.
// Default is Never.
func (p *PodBuilder) RestartPolicy(policy string) *PodBuilder {
	switch policy {
	case string(v1.RestartPolicyAlways), string(v1.RestartPolicyNever), string(v1.RestartPolicyOnFailure):
		p.pod.Spec.RestartPolicy = v1.RestartPolicy(policy)
	default:
		p.pod.Spec.RestartPolicy = v1.RestartPolicyNever
	}
	return p
}

// Build returns the prepared Pod instance
func (p *PodBuilder) Build() *v1.Pod {
	return p.pod
}

//
// ContainerBuilder
//
type ContainerBuilder struct {
	c v1.Container
}

// WithCommands set the list of commands to use with the new container
func (cb *ContainerBuilder) WithCommands(commands ...string) *ContainerBuilder {
	cb.c.Command = commands
	return cb
}

// AddArgs appends a given list of arguments to the existing
func (cb *ContainerBuilder) AddArgs(args ...string) *ContainerBuilder {
	if cb.c.Args == nil {
		cb.c.Args = []string{}
	}
	cb.c.Args = append(cb.c.Args, args...)
	return cb
}

// EnvVar sets an environment variable into the container
func (cb *ContainerBuilder) EnvVar(variable, value string) *ContainerBuilder {
	if cb.c.Env == nil {
		cb.c.Env = []v1.EnvVar{}
	}
	cb.c.Env = append(cb.c.Env, v1.EnvVar{
		Name:      variable,
		Value:     value,
	})
	return cb
}

// ImagePullPolicy sets the ImagePullPolicy for the given container.
// Default is PullAlways.
func (cb *ContainerBuilder) ImagePullPolicy(policy string) *ContainerBuilder {
	switch policy {
	case string(v1.PullIfNotPresent), string(v1.PullNever), string(v1.PullAlways):
		cb.c.ImagePullPolicy = v1.PullPolicy(policy)
	default:
		cb.c.ImagePullPolicy = v1.PullAlways
	}
	return cb
}

// AddVolumeMountConfigMapData add a VolumeMount entry to the container
// that must be related with a valid Volume defined in the Pod Spec.
func (cb *ContainerBuilder) AddVolumeMountConfigMapData(volumeName string, mountPath string, readOnly bool) *ContainerBuilder {
	if cb.c.VolumeMounts == nil {
		cb.c.VolumeMounts = []v1.VolumeMount{}
	}
	vm := v1.VolumeMount{
		Name:      volumeName,
		ReadOnly:  readOnly,
		MountPath: mountPath,
	}
	cb.c.VolumeMounts = append(cb.c.VolumeMounts, vm)
	return cb
}

// Build returns the prepared Container to be used within a Pod
func (cb *ContainerBuilder) Build() v1.Container {
	return cb.c
}

//returns whole pod log as a (meaty) string
func (c *ContextData) GetLogs(podName string) (string, error) {
	podLogOpts := v1.PodLogOptions{}
	request := c.Clients.KubeClient.CoreV1().Pods(c.Namespace).GetLogs(podName, &podLogOpts)
	podLogs, err := request.Stream()
	if err != nil {
		return "", err
	}
	defer podLogs.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func Execute(ctx1 *ContextData, command string, arguments string, podname string) (string, string, error) {
	pod, err := ctx1.Clients.KubeClient.CoreV1().Pods(ctx1.Namespace).Get(podname, metav1.GetOptions{})
	request := ctx1.Clients.KubeClient.CoreV1().RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command: []string{command, arguments},
			Stdin:   true,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(&RestConfig, "POST", request.URL())
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
	})
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed executing command %s on %v/%v", command, pod.Namespace, pod.Name)
	}
	return buf.String(), errBuf.String(), nil
}

