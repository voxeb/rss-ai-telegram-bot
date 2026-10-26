package main

import "strings"

var categoryHashtags = map[string]string{
	"ai and ml":      "#AI",
	"security":       "#Security",
	"cyber-crime":    "#Security",
	"systems":        "#Systems",
	"personal tech":  "#PersonalTech",
	"virtualization": "#Virtualization",
	"networks":       "#Networking",
	"public sector":  "#PublicSector",
	"os platforms":   "#OperatingSystems",
	"devops":         "#DevOps",
	"software":       "#Software",
	"applications":   "#Applications",
	"science":        "#Science",
	"columnists":     "#Opinion",
	"offbeat":        "#Offbeat",
	"bofh":           "#BOFH",
}

// Use the first recognized source category; unknown categories are optional metadata.
func categoryHashtag(categories []string) string {
	for _, category := range categories {
		normalized := strings.ToLower(strings.Join(strings.Fields(category), " "))
		if hashtag := categoryHashtags[normalized]; hashtag != "" {
			return hashtag
		}
	}
	return ""
}
