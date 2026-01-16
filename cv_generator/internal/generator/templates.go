package generator

import (
	"cv_generator/internal/models"
	"fmt"
	"strings"
)

// BuildCVPrompt creates the prompt for Gemini to generate a CV.
func BuildCVPrompt(data *models.ResumeData, opts *models.GenerateCVRequest) string {
	var sb strings.Builder

	// System instructions
	sb.WriteString(`You are an expert resume/CV designer. Generate a complete, professional HTML document with embedded CSS for a resume.

REQUIREMENTS:
1. Return ONLY the complete HTML document (<!DOCTYPE html> to </html>)
2. All CSS must be embedded in a <style> tag in the <head>
3. The design must be print-optimized with proper page breaks
4. Use @media print styles for PDF generation
5. Use @page rules for A4 paper size and margins
6. Add page-break-inside: avoid on sections to prevent awkward breaks
7. The output must be a single self-contained HTML file
8. DO NOT include any markdown, code blocks, or explanations - ONLY the HTML

`)

	// Style instructions
	sb.WriteString(fmt.Sprintf("STYLE: %s\n", getStyleDescription(opts.Style)))
	sb.WriteString(fmt.Sprintf("COLOR SCHEME: %s\n", getColorDescription(opts.ColorScheme)))

	if opts.IncludePhoto && data.User.AvatarURL != nil {
		sb.WriteString(fmt.Sprintf("PHOTO URL: %s (include as a circular/rounded profile photo)\n", *data.User.AvatarURL))
	} else {
		sb.WriteString("PHOTO: Do not include a photo\n")
	}

	sb.WriteString(fmt.Sprintf("LANGUAGE: Output all text in %s\n", getLanguageName(opts.Language)))

	if opts.CustomInstructions != "" {
		sb.WriteString(fmt.Sprintf("\nCUSTOM INSTRUCTIONS: %s\n", opts.CustomInstructions))
	}

	// Resume data
	sb.WriteString("\n=== RESUME DATA ===\n\n")

	// Personal info
	if data.Profile != nil {
		sb.WriteString("PERSONAL INFORMATION:\n")
		if data.Profile.FirstName != nil || data.Profile.LastName != nil {
			name := ""
			if data.Profile.FirstName != nil {
				name = *data.Profile.FirstName
			}
			if data.Profile.LastName != nil {
				name += " " + *data.Profile.LastName
			}
			sb.WriteString(fmt.Sprintf("- Name: %s\n", strings.TrimSpace(name)))
		}
		if data.Profile.Headline != nil {
			sb.WriteString(fmt.Sprintf("- Professional Title: %s\n", *data.Profile.Headline))
		}
		sb.WriteString(fmt.Sprintf("- Email: %s\n", data.User.Email))
		if data.Profile.Phone != nil {
			sb.WriteString(fmt.Sprintf("- Phone: %s\n", *data.Profile.Phone))
		}
		if data.Profile.City != nil || data.Profile.Country != nil {
			location := ""
			if data.Profile.City != nil {
				location = *data.Profile.City
			}
			if data.Profile.Country != nil {
				if location != "" {
					location += ", "
				}
				location += *data.Profile.Country
			}
			sb.WriteString(fmt.Sprintf("- Location: %s\n", location))
		}
		if data.Profile.LinkedInURL != nil {
			sb.WriteString(fmt.Sprintf("- LinkedIn: %s\n", *data.Profile.LinkedInURL))
		}
		if data.Profile.GithubURL != nil {
			sb.WriteString(fmt.Sprintf("- GitHub: %s\n", *data.Profile.GithubURL))
		}
		if data.Profile.Website != nil {
			sb.WriteString(fmt.Sprintf("- Website: %s\n", *data.Profile.Website))
		}
	}

	// Summary
	if opts.Sections.Summary && data.Profile != nil && data.Profile.Summary != nil {
		sb.WriteString(fmt.Sprintf("\nPROFESSIONAL SUMMARY:\n%s\n", *data.Profile.Summary))
	}

	// Experiences
	if opts.Sections.Experiences && len(data.Experiences) > 0 {
		sb.WriteString("\nWORK EXPERIENCE:\n")
		max := opts.MaxExperiences
		if max <= 0 || max > len(data.Experiences) {
			max = len(data.Experiences)
		}
		for i := 0; i < max; i++ {
			exp := data.Experiences[i]
			sb.WriteString(fmt.Sprintf("\n%d. %s at %s\n", i+1, exp.Title, exp.CompanyName))

			dateRange := exp.StartDate
			if exp.IsCurrent {
				dateRange += " - Present"
			} else if exp.EndDate != nil {
				dateRange += " - " + *exp.EndDate
			}
			sb.WriteString(fmt.Sprintf("   Period: %s\n", dateRange))

			if exp.Location != nil {
				sb.WriteString(fmt.Sprintf("   Location: %s\n", *exp.Location))
			}
			if exp.Description != nil {
				sb.WriteString(fmt.Sprintf("   Description: %s\n", *exp.Description))
			}
			if len(exp.Achievements) > 0 {
				sb.WriteString("   Key Achievements:\n")
				for _, ach := range exp.Achievements {
					sb.WriteString(fmt.Sprintf("   - %s\n", ach))
				}
			}
		}
	}

	// Education
	if opts.Sections.Education && len(data.Education) > 0 {
		sb.WriteString("\nEDUCATION:\n")
		max := opts.MaxEducation
		if max <= 0 || max > len(data.Education) {
			max = len(data.Education)
		}
		for i := 0; i < max; i++ {
			edu := data.Education[i]
			sb.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, edu.InstitutionName))
			if edu.Degree != nil {
				degreeStr := *edu.Degree
				if edu.FieldOfStudy != nil {
					degreeStr += " in " + *edu.FieldOfStudy
				}
				sb.WriteString(fmt.Sprintf("   Degree: %s\n", degreeStr))
			}
			if edu.StartDate != nil || edu.EndDate != nil {
				dateRange := ""
				if edu.StartDate != nil {
					dateRange = *edu.StartDate
				}
				if edu.IsCurrent {
					dateRange += " - Present"
				} else if edu.EndDate != nil {
					dateRange += " - " + *edu.EndDate
				}
				if dateRange != "" {
					sb.WriteString(fmt.Sprintf("   Period: %s\n", dateRange))
				}
			}
			if edu.Grade != nil {
				sb.WriteString(fmt.Sprintf("   Grade: %s\n", *edu.Grade))
			}
		}
	}

	// Skills
	if opts.Sections.Skills && len(data.Skills) > 0 {
		sb.WriteString("\nSKILLS:\n")

		// Group by category
		categories := make(map[string][]models.Skill)
		for _, skill := range data.Skills {
			if skill.IsCertification {
				continue // Handle separately
			}
			cat := "Other"
			if skill.Category != nil {
				cat = *skill.Category
			}
			categories[cat] = append(categories[cat], skill)
		}

		max := opts.MaxSkills
		count := 0
		for cat, skills := range categories {
			if count >= max && max > 0 {
				break
			}
			sb.WriteString(fmt.Sprintf("- %s: ", cat))
			names := []string{}
			for _, s := range skills {
				if count >= max && max > 0 {
					break
				}
				names = append(names, s.Name)
				count++
			}
			sb.WriteString(strings.Join(names, ", ") + "\n")
		}
	}

	// Certifications
	if opts.Sections.Certifications {
		var certs []models.Skill
		for _, skill := range data.Skills {
			if skill.IsCertification {
				certs = append(certs, skill)
			}
		}
		if len(certs) > 0 {
			sb.WriteString("\nCERTIFICATIONS:\n")
			for _, cert := range certs {
				sb.WriteString(fmt.Sprintf("- %s\n", cert.Name))
			}
		}
	}

	sb.WriteString("\n=== END RESUME DATA ===\n")
	sb.WriteString("\nNow generate the complete HTML document. Remember: ONLY output HTML, no markdown or explanations.")

	return sb.String()
}

func getStyleDescription(style models.CVStyle) string {
	switch style {
	case models.StyleModern:
		return "Modern - Clean, contemporary design with subtle accents, good use of whitespace, sans-serif fonts"
	case models.StyleMinimalist:
		return "Minimalist - Ultra-clean, lots of whitespace, simple typography, minimal decorations"
	case models.StyleClassic:
		return "Classic - Traditional professional resume, serif fonts, formal layout, timeless design"
	case models.StyleCreative:
		return "Creative - Bold colors, unique layouts, creative typography, stands out"
	default:
		return "Modern - Clean, contemporary design"
	}
}

func getColorDescription(scheme models.ColorScheme) string {
	switch scheme {
	case models.ColorBlue:
		return "Blue theme - Professional blue accents (#2563eb or similar), dark text, light background"
	case models.ColorGreen:
		return "Green theme - Nature-inspired green accents (#059669 or similar)"
	case models.ColorDark:
		return "Dark theme - Dark backgrounds with light text, elegant contrast"
	case models.ColorNeutral:
		return "Neutral - Grayscale, black and white, very professional"
	case models.ColorPurple:
		return "Purple theme - Creative purple accents (#7c3aed or similar)"
	case models.ColorRed:
		return "Red theme - Bold red accents (#dc2626 or similar)"
	default:
		return "Blue theme - Professional blue accents"
	}
}

func getLanguageName(code string) string {
	names := map[string]string{
		"en": "English",
		"de": "German",
		"fr": "French",
		"it": "Italian",
		"es": "Spanish",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return "English"
}
