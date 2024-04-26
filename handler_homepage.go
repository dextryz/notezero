package notezero

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

func (s *Handler) HomepageHandler(w http.ResponseWriter, r *http.Request) {

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
		fmt.Println("paging pull")
		fmt.Println(page)
	}

	data, err := s.processEmptyPrompt(r.Context(), page)
	if err != nil {
		s.log.Error("cannot process empty prompt", "error", err.Error())
	}

	// Generate view component from raw data

	component := IndexTemplate(ListArticleParams{
		Notes: data.Notes,
		Page:  page,
	})

	// 3. Render the view components on the client side.

	err = component.Render(r.Context(), w)
	if err != nil {
		s.log.Error("error rendering tmpl", "error", err.Error())
	}
}

func (s *Handler) processEmptyPrompt(ctx context.Context, page int) (*RawData, error) {

	data := RawData{}

	// 1. Pull profiles from curated list

	profileEvents, err := s.ns.pullProfileList(ctx, CURATED_LIST)
	if err != nil {
		s.log.Error("error rendering tmpl", "error", err.Error())
	}
	for _, v := range profileEvents {
		metadata, err := ParseMetadata(*v)
		if err != nil {
			s.log.Error("error rendering tmpl", "error", err.Error())
		}
		profiles[v.PubKey] = metadata
	}

	// 2. Pull next page of articles and map to profile

	noteEvents, err := s.ns.pullNextArticlePage(ctx, CURATED_LIST, page)
	if err != nil {
		s.log.Error("error rendering tmpl", "error", err.Error())
	}

	for _, v := range noteEvents {
		p, ok := profiles[v.PubKey]
		if !ok {
			s.log.Error("error rendering tmpl", "error", err.Error())
		}
		note := EnhancedEvent{
			Event:   v,
			Profile: p,
		}
		imgPath, ok := s.ns.cache.Get(v.GetID())
		if ok {
			note.ImagePath = string(imgPath)
		}
		data.Notes = append(data.Notes, note)
	}

	return &data, nil
}
