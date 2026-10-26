package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
)

type Estimate struct {
	MinTetri int64  `json:"minTetri"`
	MaxTetri int64  `json:"maxTetri"`
	Note     string `json:"note"`
	Source   string `json:"source"`
}

type pricingArea struct {
	name       string
	lat, lng   float64
	multiplier float64
}

var pricingAreas = []pricingArea{
	{"Tbilisi", 41.7151, 44.8271, 1.0},
	{"Batumi", 41.6168, 41.6367, 1.0},
	{"Kutaisi", 42.2679, 42.6946, 0.9},
	{"Rustavi", 41.5495, 44.9932, 0.9},
}

const pricingAreaRadiusKM = 20

func distanceKM(aLat, aLng, bLat, bLng float64) float64 {
	const r = 6371.0
	rad := math.Pi / 180
	dLat := (bLat - aLat) * rad
	dLng := (bLng - aLng) * rad
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(aLat*rad)*math.Cos(bLat*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * r * math.Asin(math.Sqrt(h))
}

func areaFor(p Point) (string, float64) {
	for _, a := range pricingAreas {
		if distanceKM(p.Lat, p.Lng, a.lat, a.lng) <= pricingAreaRadiusKM {
			return a.name, a.multiplier
		}
	}
	return "regions", 0.8
}

func roundTo5GEL(tetri float64) int64 {
	return int64(math.Max(500, math.Round(tetri/500)*500))
}

// ruleEstimate prices a repair from the category's typical labour and parts ranges, adjusted for area and whether
// the customer already has the parts.
func ruleEstimate(catName string, laborMin, laborMax, partsMin, partsMax int64, hasParts bool, p Point) Estimate {
	area, mult := areaFor(p)
	min, max := float64(laborMin)*mult, float64(laborMax)*mult
	partsNote := "customer already has the parts"
	if !hasParts {
		min += float64(partsMin)
		max += float64(partsMax)
		partsNote = "includes typical parts"
	}
	return Estimate{
		MinTetri: roundTo5GEL(min),
		MaxTetri: roundTo5GEL(max),
		Note:     fmt.Sprintf("Typical %s repair in %s, %s.", strings.ToLower(catName), area, partsNote),
		Source:   "rules",
	}
}

func (s *Server) estimate(ctx context.Context, categoryID int64, description string, hasParts bool, p Point) (Estimate, error) {
	var name string
	var lMin, lMax, pMin, pMax int64
	err := s.db.QueryRow(ctx, `
		SELECT name, labor_min_tetri, labor_max_tetri, parts_min_tetri, parts_max_tetri
		FROM categories WHERE id = $1 AND kind = 'repair'`, categoryID).Scan(&name, &lMin, &lMax, &pMin, &pMax)
	if err != nil {
		return Estimate{}, badRequest("Choose a valid repair category")
	}
	est := ruleEstimate(name, lMin, lMax, pMin, pMax, hasParts, p)
	if !s.ai.Enabled() || strings.TrimSpace(description) == "" {
		return est, nil
	}

	area, _ := areaFor(p)
	var out struct {
		MinGEL float64 `json:"min_gel"`
		MaxGEL float64 `json:"max_gel"`
		Note   string  `json:"note"`
	}
	prompt := fmt.Sprintf(`Category: %s
Area: %s, Georgia
Customer already has the needed parts: %t
Typical price range for this category: %d-%d GEL
Problem description: %s

Estimate the total repair cost in GEL (labour plus parts unless the customer has them). Reply as JSON {"min_gel": number, "max_gel": number, "note": "one short sentence for the customer"}.`,
		name, area, hasParts, est.MinTetri/100, est.MaxTetri/100, description)
	err = s.ai.chatJSON(ctx, "You estimate home appliance and repair prices in Georgia. Be realistic and concise.", prompt, &out)
	if err != nil || out.MinGEL <= 0 || out.MaxGEL < out.MinGEL {
		if err != nil {
			log.Printf("AI estimate failed, using rules: %v", err)
		}
		return est, nil
	}
	// Keep the AI answer within a sane band around the rule-based range.
	lo, hi := float64(est.MinTetri)/2, float64(est.MaxTetri)*2
	clamp := func(v float64) float64 { return math.Min(hi, math.Max(lo, v)) }
	return Estimate{
		MinTetri: roundTo5GEL(clamp(out.MinGEL * 100)),
		MaxTetri: roundTo5GEL(clamp(out.MaxGEL * 100)),
		Note:     strings.TrimSpace(out.Note),
		Source:   "ai",
	}, nil
}

func (s *Server) previewEstimate(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		CategoryID  int64  `json:"categoryId"`
		Description string `json:"description"`
		HasParts    bool   `json:"hasParts"`
		Location    *Point `json:"location"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := in.Location.validate(); err != nil {
		return err
	}
	est, err := s.estimate(r.Context(), in.CategoryID, in.Description, in.HasParts, *in.Location)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, est)
	return nil
}

type duplicateResult struct {
	postID int64
	reason string
}

// findDuplicate looks for an earlier post by the same customer describing the same problem.
func (s *Server) findDuplicate(ctx context.Context, customerID, categoryID int64, text string) (*duplicateResult, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, category_id, title || ' ' || description, similarity(title || ' ' || description, $3)
		FROM repair_posts
		WHERE customer_id = $1 AND created_at > now() - interval '90 days'
		ORDER BY (category_id = $2) DESC, similarity(title || ' ' || description, $3) DESC
		LIMIT 3`, customerID, categoryID, text)
	if err != nil {
		return nil, err
	}
	type candidate struct {
		id, categoryID int64
		text           string
		sim            float64
	}
	var cands []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.categoryID, &c.text, &c.sim); err != nil {
			return nil, err
		}
		cands = append(cands, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, c := range cands {
		sameCat := c.categoryID == categoryID
		if s.ai.Enabled() && sameCat && c.sim >= 0.15 {
			var out struct {
				Duplicate bool   `json:"duplicate"`
				Reason    string `json:"reason"`
			}
			err := s.ai.chatJSON(ctx,
				"You detect duplicate repair requests on a marketplace. Two posts are duplicates if they ask to fix the same problem with the same item.",
				fmt.Sprintf("Earlier post: %s\n\nNew post: %s\n\nReply as JSON {\"duplicate\": true|false, \"reason\": \"short reason\"}.", c.text, text),
				&out)
			if err == nil {
				if out.Duplicate {
					return &duplicateResult{c.id, "AI: " + strings.TrimSpace(out.Reason)}, nil
				}
				continue
			}
			log.Printf("AI duplicate check failed, using similarity: %v", err)
		}
		if (sameCat && c.sim >= 0.4) || c.sim >= 0.6 {
			return &duplicateResult{c.id, fmt.Sprintf("Very similar to your post #%d (%.0f%% text match)", c.id, c.sim*100)}, nil
		}
	}
	return nil, nil
}

// typicalEstimate is the public, rules-only price range for a repair category in Tbilisi (parts included).
func (s *Server) typicalEstimate(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.ParseInt(r.URL.Query().Get("category"), 10, 64)
	if err != nil || id <= 0 {
		return badRequest("Choose a valid repair category")
	}
	tbilisi := pricingAreas[0]
	est, err := s.estimate(r.Context(), id, "", false, Point{Lat: tbilisi.lat, Lng: tbilisi.lng})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, est)
	return nil
}
