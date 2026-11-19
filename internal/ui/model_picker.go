package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mehran9675/pocket-agent/internal/llm"
	"github.com/mehran9675/pocket-agent/internal/store"
	"github.com/mehran9675/pocket-agent/internal/tools"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
	"github.com/xmhrnx/giode"
)

// openModelSheet lets the user pick the model for this chat.
func (a *App) openModelSheet() {
	search := a.input("models/search", tr("Search models"), "")
	search.SetText("")
	a.pickerProvider = ""
	for _, p := range a.settings.Providers {
		pre, _ := llm.PresetByID(p.Preset)
		if p.Enabled && len(p.Models) == 0 && (p.APIKey != "" || pre.KeyOptional) {
			a.refreshModels(p)
		}
	}
	sh := a.openSheet("models", tr("Choose a model"), func() widgets.El { return a.modelList(search.Text()) })
	sh.footer = func() widgets.El {
		return widgets.Row(widgets.St{Gap: 10, Width: -1},
			widgets.Button("models/providers", tr("Providers"), "key-round", widgets.BtnSecondary, func() {
				a.closeSheet()
				a.navigate("settings/providers")
			}),
			widgets.Button("models/library", tr("On-device"), "hard-drive", widgets.BtnSecondary, func() {
				a.closeSheet()
				a.navigate("library")
			}),
		)
	}
}

func (a *App) modelList(query string) widgets.El {
	q := strings.ToLower(strings.TrimSpace(query))
	cur := a.currentModel()
	search := a.input("models/search", tr("Search models"), "")
	kids := []widgets.El{giode.Box(widgets.St{Width: -1, Padding: giode.Inset{Bottom: 4}}, search)}

	pick := func(ref store.ModelRef) {
		a.setModel(ref)
		a.closeSheet()
	}
	modelRow := func(ref store.ModelRef, name, sub string, badges ...widgets.El) widgets.El {
		selected := ref == cur
		return widgets.Tap("models/"+ref.Provider+"/"+ref.Model, name, func(s giode.PressState) widgets.El {
			bg := widgets.HoverBg(s, "transparent")
			if selected {
				bg = theme.T.PrimarySoft
			}
			right := []widgets.El{}
			right = append(right, badges...)
			if selected {
				right = append(right, widgets.Icon("check", 18, theme.T.Primary))
			}
			texts := []widgets.El{widgets.Line(name, theme.SizeCallout, theme.T.Text, giode.FontWeightMedium)}
			if sub != "" {
				texts = append(texts, widgets.Line(sub, theme.SizeCaption, theme.T.Muted))
			}
			return widgets.Row(widgets.St{Gap: 10, Padding: giode.SymmetricInset(12, 10), BorderRadius: 12, Background: bg, Width: -1},
				append([]widgets.El{widgets.Col(widgets.St{Gap: 2, FlexGrow: 1}, texts...)}, right...)...)
		}, func() { pick(ref) })
	}
	match := func(s ...string) bool { return matchesAny(q, s...) }
	if a.pickerProvider != "" {
		if p := a.settings.Provider(a.pickerProvider); p != nil {
			return a.providerModels(p, q, kids, modelRow)
		}
		a.pickerProvider = ""
	}

	// On-device
	var localRows []widgets.El
	if a.engine.Available() {
		for _, m := range a.models.Installed() {
			if !match(m.Name, m.File) {
				continue
			}
			var bs []widgets.El
			if m.Reasoning {
				bs = append(bs, widgets.Badge(tr("Thinking"), theme.T.Text2, theme.T.Surface2))
			}
			localRows = append(localRows, modelRow(store.ModelRef{Provider: store.LocalProviderID, Model: m.File}, m.Name, tr("{size} · private, offline", "size", tools.HumanBytes(m.Size)), bs...))
		}
	}
	if len(localRows) > 0 || q == "" {
		kids = append(kids, providerHeader("cpu", "#34d399", tr("On this device"), ""))
		if len(localRows) == 0 {
			sub := tr("Download a model to chat offline")
			if !a.engine.Available() {
				sub = tr("Not available on this device")
				if a.runtime.Status().Supported {
					sub = tr("Install the engine and a model")
				}
			}
			localRows = append(localRows, widgets.ListRow("models/getlocal", "download", "#34d399", tr("Get an on-device model"), sub, nil, func() {
				a.closeSheet()
				a.navigate("library")
			}))
		}
		kids = append(kids, widgets.Col(widgets.St{Gap: 2, Width: -1}, localRows...))
	}

	for _, p := range a.settings.Providers {
		if !p.Enabled {
			continue
		}
		p := p
		pre, _ := llm.PresetByID(p.Preset)
		models := p.Models
		if len(models) == 0 {
			models = pre.Models
		}
		var rows []widgets.El
		pinned := map[string]bool{}
		for _, id := range p.Pinned {
			pinned[id] = true
		}
		// Pinned first, then the rest; long lists are capped unless searching.
		ordered := make([]llm.ModelInfo, 0, len(models))
		for _, m := range models {
			if pinned[m.ID] {
				ordered = append(ordered, m)
			}
		}
		for _, m := range models {
			if !pinned[m.ID] {
				ordered = append(ordered, m)
			}
		}
		shown := 0
		for _, m := range ordered {
			name := m.Name
			if name == "" {
				name = m.ID
			}
			if !match(name, m.ID, p.Name) {
				continue
			}
			if q == "" && shown >= pickerPreview && !pinned[m.ID] && cur.Model != m.ID {
				continue
			}
			shown++
			sub := modelSubtitle(m, name)
			var bs []widgets.El
			if pinned[m.ID] {
				bs = append(bs, widgets.Icon("star", 14, theme.T.Warning))
			}
			rows = append(rows, modelRow(store.ModelRef{Provider: p.ID, Model: m.ID}, name, sub, bs...))
		}
		if len(rows) == 0 && q != "" {
			continue
		}
		status := ""
		if a.refresh[p.ID] {
			status = tr("Loading…")
		}
		// The rest of a long list is on the provider's own page.
		if len(models) > len(rows) {
			id := p.ID
			rows = append(rows, widgets.ListRow("models/all/"+id, "list-checks", theme.T.Muted, trn("See all {n} models", len(models)), "", nil, func() {
				a.pickerProvider = id
			}))
		}
		kids = append(kids, providerHeader(providerIcon(p.Preset), providerColor(p.Preset), p.Name, status))
		if len(rows) == 0 {
			rows = append(rows, giode.Box(widgets.St{Padding: giode.SymmetricInset(12, 8)}, widgets.Text(tr("No models yet - check the provider's settings."), theme.SizeFootnote, theme.T.Muted)))
		}
		kids = append(kids, widgets.Col(widgets.St{Gap: 2, Width: -1}, rows...))
	}
	if len(a.settings.Providers) == 0 {
		kids = append(kids, widgets.Gap(0, 8), widgets.Card(widgets.St{},
			widgets.Text(tr("No cloud providers yet"), theme.SizeCallout, theme.T.Text, giode.FontWeightSemiBold),
			widgets.Text(tr("Add an API key for Anthropic, OpenAI, Google, OpenRouter and others to use their models here."), theme.SizeFootnote, theme.T.Muted),
			widgets.Button("models/addprov", tr("Add a provider"), "plus", widgets.BtnPrimary, func() {
				a.closeSheet()
				a.navigate("settings/providers")
			})))
	}
	return widgets.Col(widgets.St{Gap: 6, Width: -1}, kids...)
}

// pickerPreview is how many of a provider's models the picker lists
// before "See all".
const pickerPreview = 6

// matchesAny reports whether any of s contains the lower-case query q.
func matchesAny(q string, s ...string) bool {
	if q == "" {
		return true
	}
	for _, x := range s {
		if strings.Contains(strings.ToLower(x), q) {
			return true
		}
	}
	return false
}

// modelSubtitle is a model's ID, when its name differs, and context size.
func modelSubtitle(m llm.ModelInfo, name string) string {
	sub := m.ID
	if sub == name {
		sub = ""
	}
	if m.ContextWindow > 0 {
		if sub != "" {
			sub += " · "
		}
		sub += fmt.Sprintf(tr("%s context"), compactNum(m.ContextWindow))
	}
	return sub
}

// providerModels is the picker's page for one provider: every model,
// searchable, favorites first.
func (a *App) providerModels(p *store.ProviderConfig, q string, kids []widgets.El, row func(store.ModelRef, string, string, ...widgets.El) widgets.El) widgets.El {
	pre, _ := llm.PresetByID(p.Preset)
	models := p.Models
	if len(models) == 0 {
		models = pre.Models
	}
	pinned := map[string]bool{}
	for _, id := range p.Pinned {
		pinned[id] = true
	}
	ordered := append([]llm.ModelInfo(nil), models...)
	sort.SliceStable(ordered, func(i, j int) bool { return pinned[ordered[i].ID] && !pinned[ordered[j].ID] })
	header := widgets.Row(widgets.St{Gap: 6, Width: -1, Align: giode.AlignCenter},
		widgets.IconButton("models/back", "arrow-left", tr("Back"), 18, theme.T.Text, func() { a.pickerProvider = "" }),
		widgets.Icon(providerIcon(p.Preset), 16, providerColor(p.Preset)),
		widgets.Text(p.Name, theme.SizeHeadline, theme.T.Text, giode.FontWeightSemiBold),
		widgets.Spacer(),
		widgets.Text(trn("{n} models", len(models)), theme.SizeCaption, theme.T.Faint))
	kids = append([]widgets.El{header}, kids...)
	var rows []widgets.El
	for _, m := range ordered {
		name := firstNonEmpty(m.Name, m.ID)
		if !matchesAny(q, name, m.ID) {
			continue
		}
		var bs []widgets.El
		if pinned[m.ID] {
			bs = append(bs, widgets.Icon("star", 14, theme.T.Warning))
		}
		rows = append(rows, row(store.ModelRef{Provider: p.ID, Model: m.ID}, name, modelSubtitle(m, name), bs...))
	}
	if len(rows) == 0 {
		rows = append(rows, giode.Box(widgets.St{Padding: giode.SymmetricInset(12, 8)}, widgets.Text(tr("No models match."), theme.SizeFootnote, theme.T.Muted)))
	}
	kids = append(kids, widgets.Col(widgets.St{Gap: 2, Width: -1}, rows...))
	return widgets.Col(widgets.St{Gap: 6, Width: -1}, kids...)
}
