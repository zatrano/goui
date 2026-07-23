package diff

type PatchOp string

const (
	OpReplace    PatchOp = "replace"
	OpUpdateText PatchOp = "update_text"
	OpSetAttr    PatchOp = "set_attr"
	OpRemoveAttr PatchOp = "remove_attr"
	OpInsert     PatchOp = "insert"
	OpRemove     PatchOp = "remove"
	OpMove       PatchOp = "move"
)

type Patch struct {
	Op      PatchOp `json:"op"`
	Path    []int   `json:"path"`
	Tag     string  `json:"tag,omitempty"`
	Text    string  `json:"text,omitempty"`
	Attr    string  `json:"attr,omitempty"`
	Value   string  `json:"value,omitempty"`
	HTML    string  `json:"html,omitempty"`
	Key     string  `json:"key,omitempty"`
	FromIdx int     `json:"from_idx,omitempty"`
	ToIdx   int     `json:"to_idx,omitempty"`
}
