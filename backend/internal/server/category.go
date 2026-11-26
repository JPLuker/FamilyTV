package server

import "strings"

func firstCategory(values []string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" && len([]rune(v)) <= 100 {
			return v
		}
	}
	return ""
}

func suggestCategory(m Metadata) string {
	// Prefer a specific publisher category. Broad platform labels offer little help on a child's board.
	category := strings.ToLower(m.Category)
	if category != "" && category != "education" && category != "entertainment" {
		return m.Category
	}
	words := " " + strings.Join(strings.FieldsFunc(strings.ToLower(m.Title+" "+m.Series+" "+m.Creator+" "+m.Playlist), func(r rune) bool { return !(r >= 'a' && r <= 'z') }), " ") + " "
	for _, rule := range []struct {
		label    string
		keywords []string
	}{
		{"Art", []string{"art", "draw", "drawing", "crafts"}}, {"Mathematics", []string{"math", "maths", "numberblocks", "counting"}},
		{"History", []string{"history", "historic", "historical"}}, {"Space", []string{"space", "artemis", "astronomy"}},
		{"Nature", []string{"wildlife", "zoo", "forest", "bbc earth", "animals", "habitats"}},
		{"Geography", []string{"geography", "mountains"}}, {"Science", []string{"science", "physics", "engineering"}},
		{"Music", []string{"songs", "music", "symphony", "orchestra"}}, {"Movement", []string{"yoga", "exercise"}},
		{"Reading", []string{"phonics", "alphabet", "alphablocks"}},
	} {
		for _, word := range rule.keywords {
			if strings.Contains(words, " "+word+" ") {
				return rule.label
			}
		}
	}
	return m.Category
}

func (s *Server) storeScanItem(item Media, existing bool) (int, error) {
	if !existing {
		return s.insertMedia(item)
	}
	for i, m := range s.data.Media {
		if m.Path != item.Path || m.CategoryChecked {
			continue
		}
		category := suggestCategory(item.Original)
		next := m
		next.CategoryChecked = true
		if next.Review.Category == "" {
			next.Review.Category = category
		}
		if next.Review.Subject == "" {
			next.Review.Subject = next.Review.Category
		}
		if next.Review.Subject != m.Review.Subject || next.Review.Category != m.Review.Category {
			next.ReviewVersion++
		}
		s.data.Media[i] = next
		if err := s.persist(); err != nil {
			s.data.Media[i] = m
			return 500, err
		}
		break
	}
	return 409, nil
}
