package diff

import "sort"

func Diff(old, new *Node) []Patch {
	var patches []Patch
	diffNode(old, new, nil, &patches)
	return patches
}

func diffNode(old, new *Node, path []int, patches *[]Patch) {
	switch {
	case old == nil && new == nil:
		return
	case old == nil:
		*patches = append(*patches, Patch{
			Op:   OpInsert,
			Path: clonePath(path),
			HTML: Serialize(new),
			Key:  new.Key,
			Tag:  new.Tag,
		})
		return
	case new == nil:
		*patches = append(*patches, Patch{
			Op:   OpRemove,
			Path: clonePath(path),
			Key:  old.Key,
			Tag:  old.Tag,
		})
		return
	}

	if old.Tag != new.Tag {
		*patches = append(*patches, Patch{
			Op:   OpReplace,
			Path: clonePath(path),
			HTML: Serialize(new),
			Key:  new.Key,
			Tag:  new.Tag,
		})
		return
	}

	if isTextNode(old) && isTextNode(new) {
		if old.Text != new.Text {
			*patches = append(*patches, Patch{
				Op:   OpUpdateText,
				Path: clonePath(path),
				Text: new.Text,
			})
		}
		return
	}

	diffAttrs(old, new, path, patches)
	diffChildren(old.Children, new.Children, path, patches)
}

func diffAttrs(old, new *Node, path []int, patches *[]Patch) {
	keys := make(map[string]struct{}, len(old.Attrs)+len(new.Attrs))
	for key := range old.Attrs {
		keys[key] = struct{}{}
	}
	for key := range new.Attrs {
		keys[key] = struct{}{}
	}

	sorted := make([]string, 0, len(keys))
	for key := range keys {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)

	for _, key := range sorted {
		oldVal, oldOK := old.Attrs[key]
		newVal, newOK := new.Attrs[key]
		switch {
		case !oldOK && newOK:
			*patches = append(*patches, Patch{
				Op:    OpSetAttr,
				Path:  clonePath(path),
				Attr:  key,
				Value: newVal,
			})
		case oldOK && !newOK:
			*patches = append(*patches, Patch{
				Op:   OpRemoveAttr,
				Path: clonePath(path),
				Attr: key,
			})
		case oldOK && newOK && oldVal != newVal:
			*patches = append(*patches, Patch{
				Op:    OpSetAttr,
				Path:  clonePath(path),
				Attr:  key,
				Value: newVal,
			})
		}
	}
}

func diffChildren(oldChildren, newChildren []*Node, path []int, patches *[]Patch) {
	if hasAnyKey(oldChildren) || hasAnyKey(newChildren) {
		diffKeyedChildren(oldChildren, newChildren, path, patches)
		return
	}
	diffIndexedChildren(oldChildren, newChildren, path, patches)
}

func diffIndexedChildren(oldChildren, newChildren []*Node, path []int, patches *[]Patch) {
	common := len(oldChildren)
	if len(newChildren) < common {
		common = len(newChildren)
	}

	for i := 0; i < common; i++ {
		childPath := append(clonePath(path), i)
		diffNode(oldChildren[i], newChildren[i], childPath, patches)
	}

	for i := common; i < len(newChildren); i++ {
		childPath := append(clonePath(path), i)
		*patches = append(*patches, Patch{
			Op:   OpInsert,
			Path: childPath,
			HTML: Serialize(newChildren[i]),
			Key:  newChildren[i].Key,
			Tag:  newChildren[i].Tag,
		})
	}

	for i := len(oldChildren) - 1; i >= common; i-- {
		childPath := append(clonePath(path), i)
		*patches = append(*patches, Patch{
			Op:   OpRemove,
			Path: childPath,
			Key:  oldChildren[i].Key,
			Tag:  oldChildren[i].Tag,
		})
	}
}

func diffKeyedChildren(oldChildren, newChildren []*Node, path []int, patches *[]Patch) {
	oldByKey := make(map[string]*Node, len(oldChildren))
	oldPos := make(map[string]int, len(oldChildren))
	newByKey := make(map[string]*Node, len(newChildren))
	newPos := make(map[string]int, len(newChildren))

	for i, child := range oldChildren {
		if child != nil && child.Key != "" {
			oldByKey[child.Key] = child
			oldPos[child.Key] = i
		}
	}
	for i, child := range newChildren {
		if child != nil && child.Key != "" {
			newByKey[child.Key] = child
			newPos[child.Key] = i
		}
	}

	for i := len(oldChildren) - 1; i >= 0; i-- {
		child := oldChildren[i]
		if child == nil || child.Key == "" {
			continue
		}
		if _, ok := newByKey[child.Key]; !ok {
			*patches = append(*patches, Patch{
				Op:   OpRemove,
				Path: append(clonePath(path), i),
				Key:  child.Key,
				Tag:  child.Tag,
			})
		}
	}

	hasInsertOrRemove := false
	for _, child := range newChildren {
		if child != nil && child.Key != "" {
			if _, ok := oldByKey[child.Key]; !ok {
				hasInsertOrRemove = true
				break
			}
		}
	}
	if !hasInsertOrRemove {
		for _, child := range oldChildren {
			if child != nil && child.Key != "" {
				if _, ok := newByKey[child.Key]; !ok {
					hasInsertOrRemove = true
					break
				}
			}
		}
	}

	for i, child := range newChildren {
		if child == nil || child.Key == "" {
			childPath := append(clonePath(path), i)
			var oldChild *Node
			if i < len(oldChildren) {
				oldChild = oldChildren[i]
			}
			if oldChild == nil {
				*patches = append(*patches, Patch{
					Op:   OpInsert,
					Path: childPath,
					HTML: Serialize(child),
					Tag:  child.Tag,
				})
				continue
			}
			diffNode(oldChild, child, childPath, patches)
			continue
		}

		oldChild, exists := oldByKey[child.Key]
		if !exists {
			*patches = append(*patches, Patch{
				Op:   OpInsert,
				Path: append(clonePath(path), i),
				HTML: Serialize(child),
				Key:  child.Key,
				Tag:  child.Tag,
			})
			continue
		}

		if !hasInsertOrRemove && oldPos[child.Key] != newPos[child.Key] {
			*patches = append(*patches, Patch{
				Op:      OpMove,
				Path:    clonePath(path),
				Key:     child.Key,
				FromIdx: oldPos[child.Key],
				ToIdx:   newPos[child.Key],
			})
		}

		diffNode(oldChild, child, append(clonePath(path), i), patches)
	}
}
