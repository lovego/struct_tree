package concise_tree

import (
	"reflect"
	"strings"

	"github.com/lovego/struct_tag"
)

type NormalTreeNode struct {
	Path     string            `json:"path"`
	Tags     map[string]string `json:"tags,omitempty"`
	Children []NormalTreeNode  `json:"children,omitempty"`
}

func (t NormalTreeNode) ChildrenPaths() []string {
	var paths []string
	for i := range t.Children {
		paths = append(paths, t.Children[i].Path)
	}
	return paths
}

func (t *NormalTreeNode) setupPathsMap(m map[string]*NormalTreeNode) {
	m[t.Path] = t
	for i := range t.Children {
		t.Children[i].setupPathsMap(m)
	}
}

func (t NormalTreeNode) keep(fn func(NormalTreeNode) bool) (NormalTreeNode, []string) {
	var tree NormalTreeNode
	var removedPaths []string
	tree.Path, tree.Tags = t.Path, t.Tags
	for _, child := range t.Children {
		if fn(child) {
			newChild, removed := child.keep(fn)
			tree.Children = append(tree.Children, newChild)
			removedPaths = append(removedPaths, removed...)
		} else {
			removedPaths = append(removedPaths, child.Path)
		}
	}
	return tree, removedPaths
}

// ExpandPath expand path so that it is not ancestor of any excluding path.
func (t *NormalTreeNode) ExpandPath(excludingPaths []string) []string {
	if !t.Contains(excludingPaths) {
		return []string{t.Path}
	}
	var result []string
	for _, child := range t.Children {
		result = append(result, child.ExpandPath(excludingPaths)...)
	}
	return result
}

func (t *NormalTreeNode) Contains(excludingPaths []string) bool {
	for _, excludingPath := range excludingPaths {
		if Contains(t.Path, excludingPath) {
			return true
		}
	}
	return false
}

// Contains return true if path equal subpath or path is ancestor of subpath.
func Contains(path, subpath string) bool {
	return strings.HasPrefix(subpath, path) &&
		(len(subpath) == len(path) || subpath[len(path)] == '.')
}

func convert(tree *NormalTreeNode, node reflect.Value) {
	if node.Kind() == reflect.Ptr {
		node = node.Elem()
	}

	typ := node.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		switch field.Type {
		case nodeType, ptr2nodeType:
			convertLeafNode(tree, field, node.Field(i))
		default:
			convertNonleafNode(tree, field, node.Field(i))
		}
	}
}

func convertLeafNode(tree *NormalTreeNode, field reflect.StructField, value reflect.Value) {
	if value.Kind() != reflect.Ptr {
		value = value.Addr()
	}
	node := value.Interface().(*Node)
	if field.Anonymous {
		tree.Path, tree.Tags = node.Path(), node.Tags()
	} else {
		tree.Children = append(tree.Children, NormalTreeNode{Path: node.Path(), Tags: node.Tags()})
	}
}

func convertNonleafNode(tree *NormalTreeNode, field reflect.StructField, value reflect.Value) {
	if field.Anonymous && struct_tag.Get(string(field.Tag), "name") == `` {
		// 匿名嵌入且节点名称为空，只用来做类型共享
		convert(tree, value)
	} else if exported(field.Name) {
		// 其余的导出字段都应该是树节点
		child := NormalTreeNode{}
		convert(&child, value)
		tree.Children = append(tree.Children, child)
	}
}
