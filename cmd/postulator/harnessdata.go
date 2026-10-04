//go:build uiharness

package main

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type callOwner int

const (
	ownedByNobody callOwner = iota
	ownedByConversation
	ownedByFinishedRun
	ownedByPastRun
)

type seedCall struct {
	Ago     time.Duration
	Owner   callOwner
	Run     int
	Item    int
	Step    string
	Model   string
	Tier    llm.ServiceTier
	Usage   llm.Usage
	Latency time.Duration
	Failed  errors.Code
}

const (
	writerModel = "gpt-5.6-terra"
	editorModel = "gpt-5.6-luna"

	day = 24 * time.Hour
)

type seedEntity struct {
	Name    string
	Kind    string
	Intent  string
	Keyword string
	Anchor  string
	Parent  string
	Path    string
	Title   string
	Status  string
}

type seedEdge struct {
	From   string
	To     string
	Status string
	Reason string
}

type seedPage struct {
	Path   string
	Title  string
	Status string
	Entity string
}

const (
	statusPublished = "published"
	statusExists    = "exists"
	statusPlanned   = "planned"
	statusArchived  = "archived"
)

func seedEntities() []seedEntity {
	return []seedEntity{
		{Name: "Espresso machines", Kind: "hub", Intent: "commercial", Keyword: "espresso machines", Anchor: "espresso machines", Path: "/espresso-machines/", Title: "Espresso machines: every type explained", Status: statusPublished},
		{Name: "Espresso machines under $500", Kind: "topic", Intent: "commercial", Keyword: "best espresso machine under 500", Anchor: "espresso machines under $500", Parent: "Espresso machines", Path: "/espresso-machines/under-500/", Title: "The best espresso machines under $500", Status: statusPublished},
		{Name: "Semi-automatic espresso machines", Kind: "topic", Intent: "commercial", Keyword: "semi automatic espresso machine", Anchor: "semi-automatic machines", Parent: "Espresso machines", Path: "/espresso-machines/semi-automatic/", Title: "Semi-automatic espresso machines, and who they suit", Status: statusPublished},
		{Name: "Super-automatic espresso machines", Kind: "topic", Intent: "commercial", Keyword: "super automatic espresso machine", Anchor: "super-automatic machines", Parent: "Espresso machines", Path: "/espresso-machines/super-automatic/", Title: "Super-automatic espresso machines: bean to cup at home", Status: statusPublished},
		{Name: "Manual lever machines", Kind: "topic", Intent: "informational", Keyword: "manual lever espresso machine", Anchor: "lever machines", Parent: "Espresso machines", Path: "/espresso-machines/manual-lever/", Title: "Manual lever machines and the shots they pull", Status: statusExists},
		{Name: "Dual boiler machines", Kind: "topic", Intent: "commercial", Keyword: "dual boiler espresso machine", Anchor: "dual boiler machines", Parent: "Espresso machines", Path: "/espresso-machines/dual-boiler/", Title: "Dual boiler machines: steam and brew at once", Status: statusPublished},
		{Name: "Espresso machine maintenance", Kind: "topic", Intent: "informational", Keyword: "espresso machine maintenance", Anchor: "machine maintenance", Parent: "Espresso machines", Path: "/espresso-machines/maintenance/", Title: "Espresso machine maintenance, week by week", Status: statusPublished},
		{Name: "Descaling an espresso machine", Kind: "topic", Intent: "informational", Keyword: "how to descale an espresso machine", Anchor: "descaling", Parent: "Espresso machine maintenance", Path: "/espresso-machines/maintenance/descaling/", Title: "How to descale an espresso machine without ruining it", Status: statusExists},

		{Name: "Grinders", Kind: "hub", Intent: "commercial", Keyword: "espresso grinders", Anchor: "grinders", Path: "/grinders/", Title: "Espresso grinders: the part that matters most", Status: statusPublished},
		{Name: "Hand grinders", Kind: "topic", Intent: "commercial", Keyword: "hand grinder for espresso", Anchor: "hand grinders", Parent: "Grinders", Path: "/grinders/hand/", Title: "Hand grinders that can actually grind for espresso", Status: statusPublished},
		{Name: "Electric burr grinders", Kind: "topic", Intent: "commercial", Keyword: "electric burr grinder", Anchor: "electric burr grinders", Parent: "Grinders", Path: "/grinders/electric/", Title: "Electric burr grinders for the home bar", Status: statusPublished},
		{Name: "Flat burr grinders", Kind: "topic", Intent: "informational", Keyword: "flat burr grinder", Anchor: "flat burrs", Parent: "Grinders", Path: "/grinders/flat-burr/", Title: "Flat burr grinders and the cup they give you", Status: statusPlanned},
		{Name: "Conical burr grinders", Kind: "topic", Intent: "informational", Keyword: "conical burr grinder", Anchor: "conical burrs", Parent: "Grinders", Path: "/grinders/conical-burr/", Title: "Conical burr grinders: forgiving and fast", Status: statusPlanned},
		{Name: "Grind size for espresso", Kind: "topic", Intent: "informational", Keyword: "espresso grind size", Anchor: "grind size", Parent: "Grinders", Path: "/grinders/grind-size/", Title: "Grind size for espresso, and how to read the shot", Status: statusPublished},
		{Name: "Single dosing", Kind: "topic", Intent: "informational", Keyword: "single dosing grinder", Anchor: "single dosing", Parent: "Grinders", Path: "/grinders/single-dosing/", Title: "Single dosing: weigh in, grind, walk away", Status: statusPlanned},

		{Name: "Brewing technique", Kind: "hub", Intent: "informational", Keyword: "espresso brewing technique", Anchor: "brewing technique", Path: "/brewing/", Title: "Espresso brewing technique from grind to cup", Status: statusPublished},
		{Name: "Portafilter", Kind: "category", Intent: "informational", Keyword: "portafilter", Anchor: "the portafilter", Parent: "Brewing technique", Path: "/brewing/portafilter/", Title: "The portafilter, basket by basket", Status: statusPublished},
		{Name: "Bottomless portafilter", Kind: "topic", Intent: "informational", Keyword: "bottomless portafilter", Anchor: "bottomless portafilter", Parent: "Portafilter", Path: "/brewing/portafilter/bottomless/", Title: "What a bottomless portafilter shows you", Status: statusPublished},
		{Name: "Tamping", Kind: "topic", Intent: "informational", Keyword: "how to tamp espresso", Anchor: "tamping", Parent: "Brewing technique", Path: "/brewing/tamping/", Title: "Tamping: level matters more than force", Status: statusPublished},
		{Name: "WDT distribution", Kind: "topic", Intent: "informational", Keyword: "wdt tool espresso", Anchor: "WDT", Parent: "Brewing technique", Path: "/brewing/wdt/", Title: "WDT: stirring the grounds before you tamp", Status: statusPublished},
		{Name: "Puck preparation", Kind: "topic", Intent: "informational", Keyword: "espresso puck prep", Anchor: "puck preparation", Parent: "Brewing technique", Path: "/brewing/puck-preparation/", Title: "Puck preparation, the whole routine", Status: statusExists},
		{Name: "Pre-infusion", Kind: "topic", Intent: "informational", Keyword: "espresso pre infusion", Anchor: "pre-infusion", Parent: "Brewing technique", Path: "/brewing/pre-infusion/", Title: "Pre-infusion: wetting the puck before the pressure", Status: statusPlanned},
		{Name: "Extraction ratios", Kind: "topic", Intent: "informational", Keyword: "espresso brew ratio", Anchor: "extraction ratios", Parent: "Brewing technique", Path: "/brewing/extraction-ratios/", Title: "Extraction ratios: ristretto, normale, lungo", Status: statusPublished},
		{Name: "Channeling", Kind: "topic", Intent: "informational", Keyword: "espresso channeling", Anchor: "channeling", Parent: "Brewing technique", Path: "/brewing/channeling/", Title: "Channeling: why one side of the puck runs fast", Status: statusPlanned},
		{Name: "Dialing in espresso", Kind: "topic", Intent: "informational", Keyword: "how to dial in espresso", Anchor: "dialing in", Parent: "Brewing technique", Path: "/brewing/dialing-in/", Title: "Dialing in espresso in three shots", Status: statusPublished},

		{Name: "Milk and drinks", Kind: "hub", Intent: "informational", Keyword: "espresso milk drinks", Anchor: "milk drinks", Path: "/milk-drinks/", Title: "Milk drinks built on espresso", Status: statusPublished},
		{Name: "Milk texturing", Kind: "topic", Intent: "informational", Keyword: "how to steam milk", Anchor: "milk texturing", Parent: "Milk and drinks", Path: "/milk-drinks/texturing/", Title: "Milk texturing: air first, then the whirlpool", Status: statusPublished},
		{Name: "Latte art", Kind: "topic", Intent: "informational", Keyword: "latte art for beginners", Anchor: "latte art", Parent: "Milk and drinks", Path: "/milk-drinks/latte-art/", Title: "Latte art for beginners: heart, tulip, rosetta", Status: statusPublished},
		{Name: "Cappuccino", Kind: "topic", Intent: "informational", Keyword: "cappuccino recipe", Anchor: "cappuccino", Parent: "Milk and drinks", Path: "/milk-drinks/cappuccino/", Title: "The cappuccino, as it is served in Italy", Status: statusPublished},
		{Name: "Flat white", Kind: "topic", Intent: "informational", Keyword: "flat white recipe", Anchor: "flat white", Parent: "Milk and drinks", Path: "/milk-drinks/flat-white/", Title: "The flat white and what makes it different", Status: statusExists},
		{Name: "Cortado", Kind: "topic", Intent: "informational", Keyword: "cortado recipe", Anchor: "cortado", Parent: "Milk and drinks", Path: "/milk-drinks/cortado/", Title: "The cortado: equal parts, small glass", Status: statusPlanned},

		{Name: "Coffee beans", Kind: "hub", Intent: "commercial", Keyword: "espresso beans", Anchor: "coffee beans", Path: "/beans/", Title: "Coffee beans for espresso", Status: statusPublished},
		{Name: "Espresso roast profiles", Kind: "topic", Intent: "informational", Keyword: "espresso roast profile", Anchor: "roast profiles", Parent: "Coffee beans", Path: "/beans/roast-profiles/", Title: "Espresso roast profiles from light to dark", Status: statusPublished},
		{Name: "Single origin espresso", Kind: "topic", Intent: "commercial", Keyword: "single origin espresso", Anchor: "single origin", Parent: "Coffee beans", Path: "/beans/single-origin/", Title: "Single origin espresso: sweet, sharp, seasonal", Status: statusPlanned},
		{Name: "Espresso blends", Kind: "topic", Intent: "commercial", Keyword: "espresso blend", Anchor: "espresso blends", Parent: "Coffee beans", Path: "/beans/blends/", Title: "Espresso blends and why roasters build them", Status: statusPlanned},
		{Name: "Coffee freshness", Kind: "topic", Intent: "informational", Keyword: "coffee degassing time", Anchor: "coffee freshness", Parent: "Coffee beans", Path: "/beans/freshness/", Title: "Coffee freshness: rest it, then drink it", Status: statusPlanned},

		{Name: "Accessories", Kind: "hub", Intent: "transactional", Keyword: "espresso accessories", Anchor: "accessories", Path: "/accessories/", Title: "Espresso accessories worth the counter space", Status: statusPublished},
		{Name: "Tampers", Kind: "product", Intent: "transactional", Keyword: "espresso tamper", Anchor: "tampers", Parent: "Accessories", Path: "/accessories/tampers/", Title: "Tampers: 58.5mm, flat, and heavy enough", Status: statusPublished},
		{Name: "Distribution tools", Kind: "product", Intent: "transactional", Keyword: "espresso distribution tool", Anchor: "distribution tools", Parent: "Accessories", Path: "/accessories/distribution-tools/", Title: "Distribution tools that earn their price", Status: statusExists},
		{Name: "Espresso scales", Kind: "product", Intent: "transactional", Keyword: "espresso scale", Anchor: "espresso scales", Parent: "Accessories", Path: "/accessories/scales/", Title: "Espresso scales with a timer that keeps up", Status: statusPublished},
		{Name: "Knock boxes", Kind: "product", Intent: "transactional", Keyword: "espresso knock box", Anchor: "knock boxes", Parent: "Accessories", Path: "/accessories/knock-boxes/", Title: "Knock boxes that stay quiet on the bench", Status: statusPlanned},

		{Name: "Home roasting", Kind: "hub", Intent: "informational", Keyword: "home coffee roasting", Anchor: "home roasting", Path: "/home-roasting/", Title: "Home roasting: green beans to a first crack", Status: statusPlanned},
		{Name: "Roasting drums", Kind: "topic", Intent: "informational", Keyword: "home roasting drum", Anchor: "roasting drums", Parent: "Home roasting", Path: "/home-roasting/drums/", Title: "Roasting drums that fit on a kitchen hob", Status: statusPlanned},
	}
}

func seedRelated() []seedEdge {
	return []seedEdge{
		{From: "Grind size for espresso", To: "Dialing in espresso", Status: "approved", Reason: "A grind change is the first move when a shot is dialed in."},
		{From: "Tamping", To: "Puck preparation", Status: "approved", Reason: "Tamping is the last step of the puck preparation routine."},
		{From: "WDT distribution", To: "Channeling", Status: "approved", Reason: "Stirring the grounds is what removes the clumps that channel."},
		{From: "Milk texturing", To: "Latte art", Status: "approved", Reason: "Latte art is poured out of textured milk, never into it."},
		{From: "Hand grinders", To: "Single dosing", Status: "approved", Reason: "A hand grinder is single dosed by construction."},
		{From: "Espresso machines under $500", To: "Semi-automatic espresso machines", Status: "proposed", Reason: "Nearly every machine under $500 is semi-automatic."},
		{From: "Pre-infusion", To: "Channeling", Status: "proposed", Reason: "A slow pre-infusion settles the puck and reduces channeling."},
		{From: "Extraction ratios", To: "Espresso roast profiles", Status: "proposed", Reason: "A darker roast is usually pulled at a longer ratio."},
		{From: "Flat burr grinders", To: "Conical burr grinders", Status: "proposed", Reason: "The two burr shapes are the choice a buyer actually makes."},
		{From: "Descaling an espresso machine", To: "Espresso machine maintenance", Status: "proposed", Reason: "Descaling is one item on the maintenance calendar."},
		{From: "Espresso scales", To: "Extraction ratios", Status: "proposed", Reason: "A ratio can only be held with a scale under the cup."},
	}
}

func seedSecondaryPages() []seedPage {
	return []seedPage{
		{Path: "/breville-bambino-plus-review/", Title: "Breville Bambino Plus review: a year on the counter", Status: statusPublished, Entity: "Espresso machines under $500"},
		{Path: "/1zpresso-jx-pro-review/", Title: "1Zpresso JX-Pro review: espresso from a hand grinder", Status: statusPublished, Entity: "Hand grinders"},
		{Path: "/rosetta-tutorial/", Title: "Pouring a rosetta: the wiggle, the cut, the finish", Status: statusPublished, Entity: "Latte art"},
	}
}

func seedLoosePages() []seedPage {
	return []seedPage{
		{Path: "/blog/", Title: "The espresso journal", Status: statusPublished},
		{Path: "/blog/espresso-for-beginners/", Title: "Espresso for beginners: your first two weeks", Status: statusPublished},
		{Path: "/blog/why-your-shot-tastes-sour/", Title: "Why your shot tastes sour", Status: statusPublished},
		{Path: "/blog/water-hardness-and-espresso/", Title: "Water hardness and espresso", Status: statusPublished},
		{Path: "/blog/a-morning-at-a-trieste-roastery/", Title: "A morning at a Trieste roastery", Status: statusPublished},
		{Path: "/blog/the-2025-buying-guide/", Title: "The 2025 buying guide", Status: statusArchived},
		{Path: "/reviews/", Title: "Machine reviews", Status: statusPublished},
		{Path: "/reviews/gaggia-classic-pro/", Title: "Gaggia Classic Pro review", Status: statusPublished},
		{Path: "/reviews/rancilio-silvia/", Title: "Rancilio Silvia review", Status: statusExists},
		{Path: "/reviews/lelit-anna/", Title: "Lelit Anna review", Status: statusExists},
		{Path: "/glossary/", Title: "An espresso glossary", Status: statusPublished},
		{Path: "/faq/", Title: "Questions we are asked every week", Status: statusPublished},
		{Path: "/newsletter/", Title: "The Thursday newsletter", Status: statusExists},
		{Path: "/about/", Title: "About this bench", Status: statusPublished},
		{Path: "/contact/", Title: "Contact", Status: statusPublished},
		{Path: "/shipping-and-returns/", Title: "Shipping and returns", Status: statusPublished},
		{Path: "/privacy-policy/", Title: "Privacy policy", Status: statusPublished},
		{Path: "/terms/", Title: "Terms of sale", Status: statusPublished},
	}
}

func spent(input, cached, output, reasoning int) llm.Usage {
	return llm.Usage{Input: input, CachedInput: cached, Output: output, Reasoning: reasoning, Total: input + output}
}

func pastRunCalls(run int, ago time.Duration, body, meta [2]llm.Usage) []seedCall {
	out := make([]seedCall, 0, 4)
	for item := range 2 {
		at := ago - time.Duration(item)*4*time.Minute
		out = append(out,
			seedCall{Ago: at, Owner: ownedByPastRun, Run: run, Item: item, Step: steps.NameGenerateBody, Model: writerModel, Tier: llm.TierFlex, Usage: body[item], Latency: 94 * time.Second},
			seedCall{Ago: at - 2*time.Minute, Owner: ownedByPastRun, Run: run, Item: item, Step: steps.NameGenerateMeta, Model: editorModel, Tier: llm.TierDefault, Usage: meta[item], Latency: 3800 * time.Millisecond},
		)
	}
	return out
}

func finishedRunCalls() []seedCall {
	out := make([]seedCall, 0, 13)
	bodies := [4]llm.Usage{spent(12400, 8100, 5600, 2400), spent(11900, 8100, 5200, 2100), spent(10300, 8100, 4800, 1900), spent(11200, 8100, 5100, 2300)}
	for item := range 4 {
		at := 9*time.Minute - time.Duration(item)*90*time.Second
		out = append(out,
			seedCall{Ago: at, Owner: ownedByFinishedRun, Item: item, Step: steps.NameGenerateBody, Model: writerModel, Tier: llm.TierFlex, Usage: bodies[item], Latency: 88 * time.Second},
			seedCall{Ago: at - 40*time.Second, Owner: ownedByFinishedRun, Item: item, Step: steps.NameGenerateMeta, Model: editorModel, Tier: llm.TierDefault, Usage: spent(2300, 0, 280, 110), Latency: 3600 * time.Millisecond},
			seedCall{Ago: at - 70*time.Second, Owner: ownedByFinishedRun, Item: item, Step: steps.NameJudge, Model: editorModel, Tier: llm.TierDefault, Usage: spent(6100, 1900, 460, 240), Latency: 5200 * time.Millisecond},
		)
	}
	return append(out, seedCall{
		Ago: 6 * time.Minute, Owner: ownedByFinishedRun, Item: 1, Step: steps.NameRepairLinks, Model: editorModel,
		Tier: llm.TierDefault, Usage: spent(3400, 0, 190, 80), Latency: 2900 * time.Millisecond,
	})
}

func seedSpendHistory() []seedCall {
	out := make([]seedCall, 0, 40)
	out = append(out, pastRunCalls(0, 27*day, [2]llm.Usage{spent(11800, 0, 6900, 3100), spent(12100, 7400, 5300, 2200)}, [2]llm.Usage{spent(2100, 0, 260, 120), spent(2200, 0, 240, 90)})...)
	out = append(out, pastRunCalls(1, 18*day, [2]llm.Usage{spent(12600, 7400, 5800, 2600), spent(11400, 7400, 4900, 2000)}, [2]llm.Usage{spent(2000, 0, 250, 100), spent(2150, 0, 270, 130)})...)
	out = append(out, seedCall{
		Ago: 18*day + 2*time.Minute, Owner: ownedByPastRun, Run: 1, Item: 0, Step: steps.NameGenerateBody, Model: writerModel,
		Tier: llm.TierFlex, Latency: 1200 * time.Millisecond, Failed: errors.RateLimited,
	})
	out = append(out, pastRunCalls(2, 9*day, [2]llm.Usage{spent(10900, 7400, 4700, 1800), spent(13200, 7400, 6400, 3000)}, [2]llm.Usage{spent(1900, 0, 230, 80), spent(2050, 0, 255, 110)})...)
	out = append(out, pastRunCalls(3, 4*day, [2]llm.Usage{spent(12000, 8100, 5500, 2400), spent(11600, 8100, 5000, 2100)}, [2]llm.Usage{spent(2250, 0, 265, 120), spent(2100, 0, 240, 100)})...)
	out = append(out, finishedRunCalls()...)
	return append(out,
		seedCall{Ago: 25*day + 5*time.Minute, Step: llm.StepProbe, Model: editorModel, Tier: llm.TierDefault, Latency: 600 * time.Millisecond, Failed: errors.Unauthorized},
		seedCall{Ago: 25 * day, Step: llm.StepProbe, Model: editorModel, Tier: llm.TierDefault, Usage: spent(14, 0, 6, 0), Latency: 900 * time.Millisecond},
		seedCall{Ago: 6*day + time.Minute, Owner: ownedByConversation, Step: llm.StepTitle, Model: editorModel, Tier: llm.TierDefault, Usage: spent(420, 0, 14, 0), Latency: 1100 * time.Millisecond},
		seedCall{Ago: 6 * day, Owner: ownedByConversation, Step: llm.StepChat, Model: writerModel, Tier: llm.TierDefault, Usage: spent(24800, 0, 610, 180), Latency: 7200 * time.Millisecond},
		seedCall{Ago: 3*day + 2*time.Hour, Step: llm.StepProposeFromPages, Model: editorModel, Tier: llm.TierDefault, Usage: spent(6400, 0, 2900, 900), Latency: 21 * time.Second},
		seedCall{Ago: 2 * day, Owner: ownedByConversation, Step: llm.StepChat, Model: writerModel, Tier: llm.TierDefault, Usage: spent(26300, 23900, 660, 220), Latency: 6100 * time.Millisecond},
		seedCall{Ago: 2*day - 3*time.Hour, Step: llm.StepJudge, Model: editorModel, Tier: llm.TierDefault, Usage: spent(5200, 0, 420, 190), Latency: 4800 * time.Millisecond},
		seedCall{Ago: day, Owner: ownedByConversation, Step: llm.StepChat, Model: writerModel, Tier: llm.TierDefault, Usage: spent(27100, 24600, 540, 160), Latency: 5400 * time.Millisecond},
		seedCall{Ago: 5 * time.Hour, Owner: ownedByConversation, Step: llm.StepChat, Model: writerModel, Tier: llm.TierDefault, Latency: 800 * time.Millisecond, Failed: errors.NeedsHuman},
		seedCall{Ago: 3 * time.Hour, Owner: ownedByConversation, Step: llm.StepChat, Model: writerModel, Tier: llm.TierDefault, Usage: spent(28400, 25800, 720, 240), Latency: 6800 * time.Millisecond},
	)
}
