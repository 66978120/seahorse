package device

import pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

type Device interface {
	Release() error
	GetDeviceCount() (int, error)
	GetContainerAllocateResponse(idxs []int) (*pluginapi.ContainerAllocateResponse, error)
	IsDeviceHealthy(idx int) (bool, error)
}
