package registry

import (
	"magic/internal/adapter"
	"magic/internal/adapters/antigravity"
	"magic/internal/adapters/claude"
	"magic/internal/adapters/cline"
	"magic/internal/adapters/codex"
	"magic/internal/adapters/copilot"
	"magic/internal/adapters/cursor"
	"magic/internal/adapters/dimagent"
	"magic/internal/adapters/droid"
	"magic/internal/adapters/gemini"
	"magic/internal/adapters/grok"
	"magic/internal/adapters/hermes"
	"magic/internal/adapters/iflow"
	"magic/internal/adapters/kilo"
	"magic/internal/adapters/kimicode"
	"magic/internal/adapters/kiro"
	"magic/internal/adapters/mcpuser"
	"magic/internal/adapters/mimocode"
	"magic/internal/adapters/omp"
	"magic/internal/adapters/openclaw"
	"magic/internal/adapters/opencode"
	"magic/internal/adapters/pi"
	"magic/internal/adapters/qoder"
	"magic/internal/adapters/qwencode"
	"magic/internal/adapters/trae"
	"magic/internal/adapters/workbuddy"
	"magic/internal/adapters/zcode"
	"magic/internal/adapters/zed"
)

var all []adapter.Adapter

func init() {
	all = []adapter.Adapter{
		pi.New(),
		grok.New(),
		claude.New(),
		codex.New(),
		mcpuser.New(),
		opencode.New(),
		gemini.New(),
		copilot.New(),
		workbuddy.New(),
		trae.New(),
		cursor.New(),
		antigravity.New(),
		kimicode.New(),
		qoder.New(),
		qoder.NewCN(),
		cline.New(),
		kiro.New(),
		droid.New(),
		zcode.New(),
		qwencode.New(),
		iflow.New(),
		kilo.New(),
		zed.New(),
		omp.New(),
		hermes.New(),
		mimocode.New(),
		dimagent.New(),
		openclaw.New(),
	}
}

func List() []adapter.Adapter { return all }

func ByID(id string) (adapter.Adapter, bool) {
	for _, a := range all {
		if string(a.ID()) == id {
			return a, true
		}
	}
	return nil, false
}

func Implemented() []adapter.Adapter {
	var out []adapter.Adapter
	for _, a := range all {
		if a.Implemented() {
			out = append(out, a)
		}
	}
	return out
}
