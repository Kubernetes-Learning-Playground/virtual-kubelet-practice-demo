package pod

import (
	"context"
	"github.com/practice/virtual-kubelet-practice/pkg/remote"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	v1 "k8s.io/api/core/v1"
	"os"
	"path/filepath"
)

func (p *Provider) createPod(ctx context.Context, pod *v1.Pod) error {

	var attempt uint32 // TODO: Track attempts. Currently always 0
	logPath := filepath.Join(p.podLogRoot, string(pod.UID))
	volPath := filepath.Join(p.podVolRoot, string(pod.UID))
	err := p.refreshNodeState(ctx)
	if err != nil {
		return err
	}
	pConfig, err := remote.GeneratePodSandboxConfig(ctx, pod, logPath, attempt)
	if err != nil {
		return err
	}
	existing := p.findPodByName(pod.Namespace, pod.Name)

	// TODO: Is re-using an existing sandbox with the UID the correct behavior?
	// TODO: Should delete the sandbox if container creation fails
	var pId string
	if existing == nil {
		err = os.MkdirAll(logPath, 0755)
		if err != nil {
			return err
		}
		err = os.MkdirAll(volPath, 0755)
		if err != nil {
			return err
		}
		// TODO: Is there a race here?
		pId, err = remote.RunPodSandbox(ctx, p.runtimeClient, pConfig)
		if err != nil {
			return err
		}
	} else {
		pId = existing.status.Metadata.Uid
	}

	for _, c := range pod.Spec.Containers {
		log.G(ctx).Debugf("Pulling image %s", c.Image)
		imageRef, err := pullImage(ctx, p.imageClient, c.Image)
		if err != nil {
			return err
		}
		log.G(ctx).Debugf("Creating container %s", c.Name)
		cConfig, err := generateContainerConfig(ctx, &c, pod, imageRef, volPath, p.resourceManager, attempt)
		if err != nil {
			return err
		}
		cId, err := createContainer(ctx, p.runtimeClient, cConfig, pConfig, pId)
		if err != nil {
			return err
		}
		log.G(ctx).Debugf("Starting container %s", c.Name)
		err = startContainer(ctx, p.runtimeClient, cId)
	}

	return err
}

func (p *Provider) deletePod(ctx context.Context, pod *v1.Pod) error {
	log.G(ctx).Debugf("receive DeletePod %q", pod.Name)

	err := p.refreshNodeState(ctx)
	if err != nil {
		return err
	}

	ps, ok := p.podStatus[pod.UID]
	if !ok {
		return errdefs.NotFoundf("Pod %s not found", pod.UID)
	}

	// TODO: Check pod status for running state
	err = stopPodSandbox(ctx, p.runtimeClient, ps.status.Id)
	if err != nil {
		// Note the error, but shouldn't prevent us trying to delete
		log.G(ctx).Debug(err)
	}

	// Remove any emptyDir volumes
	// TODO: Is there other cleanup that needs to happen here?
	err = os.RemoveAll(filepath.Join(p.podVolRoot, string(pod.UID)))
	if err != nil {
		log.G(ctx).Debug(err)
	}
	err = removePodSandbox(ctx, p.runtimeClient, ps.status.Id)

	p.notifyStatus(pod)
	return err
}

func (p *Provider) getPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {

	err := p.refreshNodeState(ctx)
	if err != nil {
		return nil, err
	}

	pod := p.findPodByName(namespace, name)
	if pod == nil {
		return nil, errdefs.NotFoundf("Pod %s in namespace %s could not be found on the node", name, namespace)
	}

	return createPodSpecFromCRI(pod, p.nodeName), nil
}

// Find a pod by name and namespace. Pods are indexed by UID
func (p *Provider) findPodByName(namespace, name string) *CRIPod {
	var found *CRIPod

	for _, pod := range p.podStatus {
		if pod.status.Metadata.Name == name && pod.status.Metadata.Namespace == namespace {
			found = &pod
			break
		}
	}
	return found
}

func (p *Provider) getPodStatus(ctx context.Context, namespace, name string) (*v1.PodStatus, error) {
	log.G(ctx).Debugf("receive GetPodStatus %q", name)

	err := p.refreshNodeState(ctx)
	if err != nil {
		return nil, err
	}

	pod := p.findPodByName(namespace, name)
	if pod == nil {
		return nil, errdefs.NotFoundf("pod %s in namespace %s could not be found on the node", name, namespace)
	}

	return createPodStatusFromCRI(pod), nil
}
