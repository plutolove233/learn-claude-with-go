package ui

type Theme struct {
	Claude          string
	ClaudeShimmer   string
	Text            string
	Subtle          string
	Inactive        string
	Suggestion      string
	Warning         string
	Error           string
	Success         string
	DiffAdded       string
	DiffRemoved     string
	DiffAddedWord   string
	DiffRemovedWord string
	InlineCode      string
	QuoteBar        string
	UserBorder      string
}

func DefaultLightTheme() Theme {
	return Theme{
		Claude:          "38;2;215;119;87",
		ClaudeShimmer:   "38;2;245;149;117",
		Text:            "38;2;0;0;0",
		Subtle:          "38;2;175;175;175",
		Inactive:        "38;2;102;102;102",
		Suggestion:      "38;2;87;105;247",
		Warning:         "38;2;150;108;30",
		Error:           "38;2;171;43;63",
		Success:         "38;2;44;122;57",
		DiffAdded:       "38;2;47;157;68",
		DiffRemoved:     "38;2;209;69;75",
		DiffAddedWord:   "48;2;214;246;220;38;2;30;90;42",
		DiffRemovedWord: "48;2;252;220;226;38;2;120;30;42",
		InlineCode:      "38;2;87;105;247",
		QuoteBar:        "38;2;142;142;142",
		UserBorder:      "38;2;153;153;153",
	}
}
