package wptest

import (
	"io"
	"mime"
	"net/http"
	"path"
	"slices"
)

type Upload struct {
	Filename string
	MimeType string
	Alt      string
	Title    string
	Bytes    []byte
	ID       int64
}

func (s *Server) routeMedia(mux *http.ServeMux) {
	mux.HandleFunc("POST "+coreNamespace+"/media", s.handleMediaUpload)
	mux.HandleFunc("GET "+coreNamespace+"/media/{id}", s.handleMediaGet)
	mux.HandleFunc("POST "+coreNamespace+"/media/{id}", s.handleMediaUpdate)
}

func (s *Server) Uploads() []Upload {
	s.mu.Lock()
	defer s.mu.Unlock()

	uploads := make([]Upload, 0, len(s.uploadOrder))
	for _, id := range s.uploadOrder {
		stored := s.uploads[id]
		uploads = append(uploads, Upload{
			ID:       stored.ID,
			Filename: stored.Filename,
			MimeType: stored.MimeType,
			Alt:      stored.Alt,
			Title:    stored.Title,
			Bytes:    slices.Clone(stored.Bytes),
		})
	}
	return uploads
}

func (s *Server) handleMediaUpload(w http.ResponseWriter, r *http.Request) {
	filename := dispositionFilename(r.Header.Get("Content-Disposition"))
	if filename == "" {
		s.fail(w, http.StatusBadRequest, "rest_upload_no_content_disposition", "No Content-Disposition supplied.")
		return
	}

	payload, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes))
	if err != nil || len(payload) == 0 {
		s.fail(w, http.StatusBadRequest, "rest_upload_no_data", "No data supplied.")
		return
	}

	s.mu.Lock()
	s.nextID++
	stored := &upload{
		ID:       s.nextID,
		Filename: filename,
		MimeType: r.Header.Get("Content-Type"),
		Title:    filename,
		Bytes:    payload,
	}
	s.uploads[stored.ID] = stored
	s.uploadOrder = append(s.uploadOrder, stored.ID)
	body := s.uploadPayload(stored)
	s.mu.Unlock()

	s.respond(w, http.StatusCreated, body)
}

func (s *Server) handleMediaGet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	s.mu.Lock()
	stored, found := s.uploads[id]
	var body map[string]any
	if found {
		body = s.uploadPayload(stored)
	}
	s.mu.Unlock()

	if body == nil {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}
	s.respond(w, http.StatusOK, body)
}

func (s *Server) handleMediaUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	attributes, ok := s.decodeBody(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	stored, found := s.uploads[id]
	if !found {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "rest_post_invalid_id", "Invalid post ID.")
		return
	}

	if value, present := attributes["alt_text"].(string); present {
		stored.Alt = value
	}
	if value, present := attributes["title"].(string); present {
		stored.Title = value
	}
	body := s.uploadPayload(stored)
	s.mu.Unlock()

	s.respond(w, http.StatusOK, body)
}

func (s *Server) uploadPayload(stored *upload) map[string]any {
	return map[string]any{
		"id":         stored.ID,
		"source_url": s.http.URL + "/wp-content/uploads/" + stored.Filename,
		"alt_text":   stored.Alt,
		"mime_type":  stored.MimeType,
		"title":      renderedField(stored.Title),
	}
}

func dispositionFilename(header string) string {
	if header == "" {
		return ""
	}

	_, params, err := mime.ParseMediaType(header)
	if err != nil {
		return ""
	}

	name := params["filename"]
	if name == "" {
		return ""
	}
	return path.Base(name)
}
