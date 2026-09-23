//go:build uiharness

package main

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
		{Path: "/espresso-machines/under-500/breville-bambino-plus-review/", Title: "Breville Bambino Plus review: a year on the counter", Status: statusPublished, Entity: "Espresso machines under $500"},
		{Path: "/grinders/hand/1zpresso-jx-pro-review/", Title: "1Zpresso JX-Pro review: espresso from a hand grinder", Status: statusPublished, Entity: "Hand grinders"},
		{Path: "/milk-drinks/latte-art/rosetta-tutorial/", Title: "Pouring a rosetta: the wiggle, the cut, the finish", Status: statusPublished, Entity: "Latte art"},
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
