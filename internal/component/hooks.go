package component

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// HookFile is what a hooks file, a settings file or a plugin manifest
// declares.
type HookFile struct {
	// Problems say why Claude Code can't read all or part of it.
	Problems []string
	Groups   []Group
	Handlers []Handler
}

// A Group is one matcher group under an event.
type Group struct {
	Event string
	// Index counts the event's groups from 1.
	Index      int
	Matcher    string
	HasMatcher bool
	Keys       []string
}

type Handler struct {
	Group
	// Index counts the group's hooks from 1.
	Index   int
	Keys    []string
	Fields  map[string]json.RawMessage
	Type    string
	Command string
	Args    []string
	// Exec reports the exec form: args set, command run without a shell.
	Exec bool
}

// Where names the group as a report shows it: PreToolUse (matcher "Bash").
func (g Group) Where() string {
	if g.HasMatcher {
		return fmt.Sprintf("%s (matcher %q)", g.Event, g.Matcher)
	}
	return g.Event
}

// Where names the hook as a report shows it: PreToolUse (matcher "Bash"), hook 2.
func (h Handler) Where() string {
	return fmt.Sprintf("%s, hook %d", h.Group.Where(), h.Index)
}

// Run is the command line the hook runs.
func (h Handler) Run() string {
	return strings.Join(append([]string{h.Command}, h.Args...), " ")
}

// String returns a string field of the hook.
func (h Handler) String(key string) (string, bool) {
	var s string
	raw, ok := h.Fields[key]
	return s, ok && json.Unmarshal(raw, &s) == nil
}

// parseHooksFile reads hooks/hooks.json or a hooks file a manifest names:
// the events sit under a top-level "hooks" key.
func parseHooksFile(content []byte) HookFile {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(content, &top); err != nil {
		return HookFile{Problems: []string{jsonProblem(content, err)}}
	}
	events, ok := top["hooks"]
	if !ok {
		return HookFile{Problems: []string{`no top-level "hooks" key: Claude Code reads a hooks file's events from {"hooks": {"<Event>": [...]}}`}}
	}
	return parseEvents(events)
}

// settingsHooks reads the hooks of a settings file, if it has any.
func settingsHooks(content []byte) (HookFile, bool) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(content, &top); err != nil {
		if bytes.Contains(content, []byte(`"hooks"`)) {
			return HookFile{Problems: []string{jsonProblem(content, err)}}, true
		}
		return HookFile{}, false
	}
	events, ok := top["hooks"]
	if !ok {
		return HookFile{}, false
	}
	return parseEvents(events), true
}

// manifestHooks reads the hooks field of a plugin manifest: a path, an
// inline object, or an array of both.
func manifestHooks(content []byte) (files []string, inline json.RawMessage) {
	var manifest struct {
		Hooks json.RawMessage `json:"hooks"`
	}
	if json.Unmarshal(content, &manifest) != nil || manifest.Hooks == nil {
		return nil, nil
	}
	entries := []json.RawMessage{manifest.Hooks}
	var list []json.RawMessage
	if json.Unmarshal(manifest.Hooks, &list) == nil {
		entries = list
	}
	var objects []json.RawMessage
	for _, entry := range entries {
		var file string
		if json.Unmarshal(entry, &file) == nil {
			files = append(files, file)
			continue
		}
		// An inline object is either the event map itself or wrapped in
		// "hooks", as in a hooks file.
		var wrapped struct {
			Hooks json.RawMessage `json:"hooks"`
		}
		if json.Unmarshal(entry, &wrapped) == nil && wrapped.Hooks != nil {
			entry = wrapped.Hooks
		}
		objects = append(objects, entry)
	}
	if len(objects) == 0 {
		return files, nil
	}
	return files, mergeObjects(objects)
}

// mergeObjects joins the inline event maps of a manifest into one.
func mergeObjects(objects []json.RawMessage) json.RawMessage {
	if len(objects) == 1 {
		return objects[0]
	}
	var b bytes.Buffer
	b.WriteByte('{')
	first := true
	for _, o := range objects {
		inner := bytes.TrimSpace(o)
		inner = bytes.TrimSpace(bytes.TrimSuffix(bytes.TrimPrefix(inner, []byte("{")), []byte("}")))
		if len(inner) == 0 {
			continue
		}
		if !first {
			b.WriteByte(',')
		}
		b.Write(inner)
		first = false
	}
	b.WriteByte('}')
	return b.Bytes()
}

func parseEvents(raw json.RawMessage) HookFile {
	var hf HookFile
	problem := func(format string, a ...any) { hf.Problems = append(hf.Problems, fmt.Sprintf(format, a...)) }
	events, fields, ok := object(raw)
	if !ok {
		problem(`"hooks" must map event names to lists of matcher groups`)
		return hf
	}
	for _, event := range events {
		var groups []json.RawMessage
		if json.Unmarshal(fields[event], &groups) != nil {
			problem("%s must be a list of matcher groups, [{\"matcher\": ..., \"hooks\": [...]}]", event)
			continue
		}
		for gi, raw := range groups {
			keys, gf, ok := object(raw)
			if !ok {
				problem("%s group %d must be an object", event, gi+1)
				continue
			}
			g := Group{Event: event, Index: gi + 1, Keys: keys}
			if m, ok := gf["matcher"]; ok {
				g.HasMatcher = true
				if json.Unmarshal(m, &g.Matcher) != nil {
					problem("%s group %d: matcher must be a string", event, gi+1)
				}
			}
			hf.Groups = append(hf.Groups, g)
			var handlers []json.RawMessage
			if gf["hooks"] == nil || json.Unmarshal(gf["hooks"], &handlers) != nil {
				problem(`%s has no "hooks" list, so it runs nothing`, g.Where())
				continue
			}
			for hi, raw := range handlers {
				keys, hfields, ok := object(raw)
				if !ok {
					problem("%s, hook %d must be an object", g.Where(), hi+1)
					continue
				}
				h := Handler{Group: g, Index: hi + 1, Keys: keys, Fields: hfields}
				json.Unmarshal(hfields["type"], &h.Type)
				json.Unmarshal(hfields["command"], &h.Command)
				if args, ok := hfields["args"]; ok {
					h.Exec = true
					json.Unmarshal(args, &h.Args)
				}
				hf.Handlers = append(hf.Handlers, h)
			}
		}
	}
	return hf
}

// object decodes a JSON object, returning its keys in file order.
func object(raw json.RawMessage) ([]string, map[string]json.RawMessage, bool) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return nil, nil, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.Token()
	var keys []string
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			break
		}
		keys = append(keys, t.(string))
		var skip json.RawMessage
		if dec.Decode(&skip) != nil {
			break
		}
	}
	return keys, fields, true
}

func jsonProblem(content []byte, err error) string {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		line := bytes.Count(content[:min(int(syntax.Offset), len(content))], []byte("\n")) + 1
		return fmt.Sprintf("not valid JSON, line %d: %v", line, syntax)
	}
	return "not valid JSON: " + err.Error()
}
