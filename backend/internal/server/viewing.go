package server

import (
	"crypto/sha256"

	"encoding/hex"
	"encoding/json"
	"io"
	"maps"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"
)

type Viewing struct {
	Rotation    string        `json:"rotation,omitempty"`
	Budget      SessionBudget `json:"budget,omitempty"`
	LastCreator string        `json:"last_creator,omitempty"`
	Profile     string        `json:"profile"`
	Mode        string        `json:"mode"`
	Deadline    int64         `json:"deadline,omitempty"`
	Current     string        `json:"current,omitempty"`
	Last        bool          `json:"last"`
	JoinHash    string        `json:"join_hash,omitempty"`
	JoinUntil   int64         `json:"join_until,omitempty"`
	CookieHash  string        `json:"cookie_hash,omitempty"`
}
type WatchItem struct {
	returning bool
	creator   string
	series    string
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Subject   string  `json:"subject"`
	Duration  float64 `json:"duration_seconds"`
	Captions  bool    `json:"captions"`
}

func watchItem(m Media) WatchItem {
	return WatchItem{ID: m.ID, Title: m.Title, Subject: m.Review.Subject, Duration: m.Duration, Captions: m.Captions, creator: strings.ToLower(strings.TrimSpace(m.Metadata.Creator)), series: strings.ToLower(strings.TrimSpace(m.Metadata.Series))}
}
func secretHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func readInput(w http.ResponseWriter, r *http.Request, value any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	d.DisallowUnknownFields()
	if d.Decode(value) != nil || d.Decode(&struct{}{}) != io.EOF {
		write(w, 400, map[string]string{"error": "Invalid request."})
		return false
	}
	return true
}
func (s *Server) profile(id string) (ChildProfile, bool) {
	for _, p := range s.data.Profiles {
		if p.ID == id {
			return p, true
		}
	}
	return ChildProfile{}, false
}
func (s *Server) setViewing(w http.ResponseWriter, id string, next *Viewing) bool {
	if s.data.Viewing == nil {
		s.data.Viewing = map[string]Viewing{}
	}
	old, had := s.data.Viewing[id]
	before := s.data.Usage
	now := time.Now()
	if old.Budget.Day != "" {
		usage := s.usageAt(id, now)
		s.data.Usage = maps.Clone(before)
		if s.data.Usage == nil {
			s.data.Usage = map[string]DailyUsage{}
		}
		s.data.Usage[id] = usage
		if next != nil && next.Budget.Day == old.Budget.Day && next.Budget.Since == old.Budget.Since {
			next.Budget.Remaining = maps.Clone(next.Budget.Remaining)
			if key := old.Budget.Category; key != "" {
				next.Budget.Remaining[key] = max(0, old.Budget.CategoryEnd-now.UnixMilli())
			}
			next.Budget.Since = now.UnixMilli()
		}
	}
	if next == nil {
		delete(s.data.Viewing, id)
	} else {
		s.data.Viewing[id] = *next
	}
	if err := s.persist(); err != nil {
		s.data.Usage = before
		if had {
			s.data.Viewing[id] = old
		} else {
			delete(s.data.Viewing, id)
		}
		write(w, 500, map[string]string{"error": "storage failure"})
		return false
	}
	return true
}
func active(v Viewing, now time.Time) bool {
	return v.Profile != "" && (v.cutoff() == 0 || v.cutoff() > now.UnixMilli())
}
func (s *Server) parentViewing(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.authorized(r) {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	id := r.PathValue("id")
	profile, ok := s.profile(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	now := time.Now()
	v := s.data.Viewing[id]
	if strings.HasSuffix(r.URL.Path, "/pair") {
		if !active(v, now) {
			write(w, 409, map[string]string{"error": "Start a viewing session first."})
			return
		}
		if r.Method == "PUT" {
			var device struct {
				Challenge string `json:"challenge"`
			}
			if !readInput(w, r, &device) {
				return
			}
			decoded, err := hex.DecodeString(device.Challenge)
			if err != nil || len(decoded) != sha256.Size {
				write(w, 400, map[string]string{"error": "Invalid TV pairing request."})
				return
			}
			hash := hex.EncodeToString(decoded)
			for otherID, other := range s.data.Viewing {
				if otherID != id && other.JoinHash == hash && other.JoinUntil > now.UnixMilli() && active(other, now) {
					write(w, 409, map[string]string{"error": "This TV is waiting for another profile."})
					return
				}
			}
			if s.pendingDeviceChallenge(hash, now) {
				write(w, 409, map[string]string{"error": "This screen already has a pending request."})
				return
			}
			v.JoinHash = hash
			v.JoinUntil = now.Add(10 * time.Minute).UnixMilli()
			if s.setViewing(w, id, &v) {
				write(w, 200, map[string]bool{"ready": true})
			}
			return
		}
		if r.Method == "DELETE" {
			v.CookieHash, v.JoinHash, v.JoinUntil = "", "", 0
			if s.setViewing(w, id, &v) {
				write(w, 200, map[string]bool{"revoked": true})
			}
			return
		}
		s.issueCode(w, id, v, now)
		return
	}
	if r.Method == "GET" {
		write(w, 200, map[string]any{"active": active(v, now), "mode": v.Mode, "deadline": v.cutoff(), "playing": v.Current != "", "last": v.Last})
		return
	}
	if r.Method == "DELETE" {
		if s.setViewing(w, id, nil) {
			write(w, 200, map[string]bool{"ended": true})
		}
		return
	}
	if r.Method == "PATCH" {
		if !active(v, now) || v.Current == "" {
			if s.setViewing(w, id, nil) {
				write(w, 200, map[string]bool{"ended": true})
			}
			return
		}
		v.Last = true
		if s.setViewing(w, id, &v) {
			write(w, 200, map[string]bool{"last": true})
		}
		return
	}
	var input struct {
		Mode     string  `json:"mode"`
		PIN      *string `json:"pin,omitempty"`
		Minutes  int     `json:"minutes"`
		FinishBy string  `json:"finish_by"`
	}
	if !readInput(w, r, &input) {
		return
	}
	if input.PIN != nil && !s.checkStartPIN(w, *input.PIN) {
		return
	}
	if active(v, now) {
		write(w, 409, map[string]string{"error": "End the active session first."})
		return
	}
	v = Viewing{Profile: id, Mode: input.Mode, Rotation: token()[:12]}
	switch input.Mode {
	case "manual":
	case "one_video":
		v.Last = true
	case "duration":
		if input.Minutes <= 0 || input.Minutes > 1440 {
			write(w, 400, map[string]string{"error": "Choose 1 to 1440 minutes."})
			return
		}
		v.Deadline = now.Add(time.Duration(input.Minutes) * time.Minute).UnixMilli()
	case "finish_by":
		t, e := time.Parse(time.RFC3339, input.FinishBy)
		if e != nil || !t.After(now) {
			write(w, 400, map[string]string{"error": "Choose a future time with a time zone."})
			return
		}
		v.Deadline = t.UnixMilli()
	default:
		write(w, 400, map[string]string{"error": "Unknown session mode."})
		return
	}
	if err := s.applyPolicy(&v, profile.Policy, now); err != nil {
		write(w, 409, map[string]string{"error": err.Error()})
		return
	}
	s.issueCode(w, id, v, now)
}
func (s *Server) issueCode(w http.ResponseWriter, id string, v Viewing, now time.Time) {
	code, ok := newPairingCode(w)
	if !ok {
		return
	}
	v.JoinHash = secretHash(code)
	v.JoinUntil = now.Add(10 * time.Minute).UnixMilli()
	if s.setViewing(w, id, &v) {
		write(w, 201, map[string]any{"code": code, "deadline": v.cutoff()})
	}
}
func (s *Server) joinViewing(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code string `json:"code"`
	}
	if !readInput(w, r, &input) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	hash := secretHash(input.Code)
	normalized := secretHash(strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(input.Code), "-", "")))
	if s.joinDevice(w, r, hash, normalized, now) {
		return
	}
	for id, v := range s.data.Viewing {
		if (v.JoinHash != hash && v.JoinHash != normalized) || !active(v, now) || v.JoinUntil <= now.UnixMilli() {
			continue
		}
		_, ok := s.profile(id)
		if !ok {
			continue
		}
		cookie := token()
		v.CookieHash = secretHash(cookie)
		v.JoinHash = ""
		v.JoinUntil = 0
		s.finishExchange(w, r, cookie, 0, func() bool { return s.setViewing(w, id, &v) })

		return
	}
	write(w, 403, map[string]string{"error": "This viewing link has expired or has already been used."})
}
func (s *Server) finishExchange(w http.ResponseWriter, r *http.Request, cookie string, maxAge int, save func() bool) {
	// A parent may open the viewer link on the same browser and hand it to a child.
	// Drop that browser's parent session in the same write as the code exchange.
	parent, _ := r.Cookie("familytv_admin")
	var parentExpiry int64
	if parent != nil {
		parentExpiry = s.data.Sessions[parent.Value]
		delete(s.data.Sessions, parent.Value)
	}
	if save() {
		if parent != nil {
			http.SetCookie(w, &http.Cookie{Name: "familytv_admin", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteStrictMode})
		}
		http.SetCookie(w, &http.Cookie{Name: "familytv_viewer", Value: cookie, MaxAge: maxAge, Path: "/api/watch", HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteStrictMode})
		write(w, 200, map[string]bool{"joined": true})
	} else if parentExpiry != 0 {
		s.data.Sessions[parent.Value] = parentExpiry
	}
}
func (s *Server) viewer(r *http.Request) (Viewing, ChildProfile, bool) {
	c, e := r.Cookie("familytv_viewer")
	if e != nil {
		return Viewing{}, ChildProfile{}, false
	}
	hash := secretHash(c.Value)
	if screen, paired := s.findScreen(hash); paired {
		id := screen.Profile
		v := s.data.Viewing[id]
		p, ok := s.profile(id)
		return v, p, ok && active(v, time.Now())
	}
	for _, v := range s.data.Viewing {
		if v.CookieHash == hash && active(v, time.Now()) {
			p, ok := s.profile(v.Profile)
			return v, p, ok
		}
	}
	return Viewing{}, ChildProfile{}, false
}
func fits(m Media, v Viewing, now time.Time) bool {
	needed := int64(math.Ceil(m.Duration * 1000))
	if left, limited := v.categoryLeft(m, now); limited && needed > left {
		return false
	}
	return v.cutoff() == 0 || needed <= v.cutoff()-now.UnixMilli()
}
func available(m Media, p ChildProfile) bool {
	if !eligible(m, p) {
		return false
	}
	f, info, err := openMedia(m.Path)
	if err != nil {
		return false
	}
	f.Close()
	if info.ModTime().UnixNano() != m.Modified || info.Size() != m.Size {
		return false
	}
	if p.Captions {
		f, _, err := openCaption(m.Path)
		if err != nil {
			return false
		}
		f.Close()
	}
	return true
}
func (s *Server) watchCatalog(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, p, ok := s.viewer(r)
	if !ok {
		write(w, 403, map[string]string{"error": "Session ended."})
		return
	}
	list := r.URL.Query().Get("list")
	marks := s.data.Bookmarks[p.ID]
	items := []WatchItem{}
	var current *WatchItem
	term := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if p.Discovery.Search == "off" {
		term = ""
	}
	now := time.Now()
	for _, m := range s.data.Media {
		if !available(m, p) {
			continue
		}
		item := watchItem(m)
		if m.ID == v.Current {
			current = &item
		}
		bookmark := marks[m.ID]
		item.returning = list == "" && bookmark.Later && bookmark.SavedDuring != v.Rotation
		if list == "favorites" && !bookmark.Favorite || list == "later" && !bookmark.Later || list == "resume" && (!p.ResumeVideos || bookmark.Position <= 0) {
			continue
		}
		if fits(m, v, now) {
			items = append(items, item)
		}
	}
	items = p.Discovery.board(items, v.Rotation, v.LastCreator)
	items = slices.DeleteFunc(items, func(item WatchItem) bool {
		mark := marks[item.ID]
		if list == "" && mark.Later && mark.SavedDuring == v.Rotation {
			return true
		}
		return term != "" && !strings.Contains(strings.ToLower(item.Title+" "+item.Subject), term)
	})
	shelf := []ShelfEntry{}
	for _, item := range items {
		if mark, ok := marks[item.ID]; ok {
			shelf = append(shelf, ShelfEntry{item, mark, true})
		}
	}
	resume := float64(0)
	if p.ResumeVideos {
		resume = marks[v.Current].Position
	}
	write(w, 200, map[string]any{"rotation": v.Rotation, "items": items, "list": list, "shelf": shelf, "resume": resume, "page": 0, "pages": 1, "discovery": p.Discovery, "current": current, "deadline": v.cutoff(), "server_time": now.UnixMilli(), "captions": p.Captions, "avatar": p.Avatar, "navigation": p.Navigation, "reduced_motion": p.ReducedMotion, "large_text": p.LargeText, "high_contrast": p.HighContrast, "ending_topic": p.EndingTopic, "quiet_ending": p.QuietEnding, "ending_music": p.EndingMusic && !p.QuietEnding})
}
func (s *Server) watchAction(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, p, ok := s.viewer(r)
	if !ok {
		write(w, 403, map[string]string{"error": "Session ended."})
		return
	}
	var input struct {
		ID       string   `json:"id"`
		Position *float64 `json:"position,omitempty"`
	}
	if !readInput(w, r, &input) {
		return
	}
	if r.URL.Path == "/api/watch/finish" || r.URL.Path == "/api/watch/leave" {
		if v.Current == "" || v.Current != input.ID {
			write(w, 409, map[string]string{"error": "Video is no longer playing."})
			return
		}
		before := s.data.Bookmarks
		bookmark := s.data.Bookmarks[p.ID][v.Current]
		if r.URL.Path == "/api/watch/finish" {
			bookmark.Position = 0
			bookmark.Later = false
			bookmark.SavedDuring = ""
		} else if input.Position != nil {
			for _, m := range s.data.Media {
				if m.ID == v.Current {
					if *input.Position < 0 || *input.Position > m.Duration {
						write(w, 400, map[string]string{"error": "Invalid playback position."})
						return
					}
					bookmark.Position = *input.Position
					break
				}
			}
		}
		s.changeBookmark(p.ID, v.Current, bookmark)
		v.Budget.Category = ""
		v.Budget.CategoryEnd = 0
		v.Current = ""
		var next *Viewing = &v
		if v.Last {
			next = nil
		}
		if s.setViewing(w, p.ID, next) {
			write(w, 200, map[string]bool{"ended": next == nil})
		} else {
			s.data.Bookmarks = before
		}
		return
	}
	if v.Current != "" && v.Current != input.ID {
		stillPlayable := false
		for _, m := range s.data.Media {
			if m.ID == v.Current && available(m, p) {
				stillPlayable = true
				break
			}
		}
		if stillPlayable {
			write(w, 409, map[string]string{"error": "Finish the current video first."})
			return
		}
		if v.Last {
			if s.setViewing(w, p.ID, nil) {
				write(w, 403, map[string]string{"error": "Session ended."})
			}
			return
		}
	}
	for _, m := range s.data.Media {
		if m.ID == input.ID && available(m, p) && (v.Current == m.ID || fits(m, v, time.Now())) {
			if v.Current != m.ID {
				left, limited := v.categoryLeft(m, time.Now())
				v.Budget.Category = ""
				v.Budget.CategoryEnd = 0
				if limited {
					v.Budget.Category = strings.ToLower(strings.TrimSpace(m.Review.Category))
					v.Budget.CategoryEnd = time.Now().UnixMilli() + left
				}
			}
			v.Current = m.ID
			v.LastCreator = watchItem(m).creator
			if s.setViewing(w, p.ID, &v) {
				write(w, 200, map[string]bool{"playing": true})
			}
			return
		}
	}
	write(w, 404, map[string]string{"error": "Video unavailable in this session."})
}
