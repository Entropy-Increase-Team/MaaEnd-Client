package maa

// 资源管理相关辅助函数

// ResourceInfo 资源信息
type ResourceInfo struct {
	Name  string
	Paths []string
}

// GetResourceInfo 获取资源信息
func (w *Wrapper) GetResourceInfo(name string) *ResourceInfo {
	if w.pi == nil {
		return nil
	}

	paths := w.pi.GetResourcePaths(name)
	if paths == nil {
		return nil
	}

	return &ResourceInfo{
		Name:  name,
		Paths: paths,
	}
}

// GetAllResources 获取所有资源
func (w *Wrapper) GetAllResources() []ResourceInfo {
	if w.pi == nil {
		return nil
	}

	names := w.pi.GetResourceNames()
	resources := make([]ResourceInfo, 0, len(names))

	for _, name := range names {
		paths := w.pi.GetResourcePaths(name)
		resources = append(resources, ResourceInfo{
			Name:  name,
			Paths: paths,
		})
	}

	return resources
}
