package core

import "errors"

// ErrSkipRender tells the session to acknowledge an event without sending a render frame.
// Use for client-owned UI (Quill, CodeMirror) where patching would destroy editor DOM.
var ErrSkipRender = errors.New("goui: skip render")
